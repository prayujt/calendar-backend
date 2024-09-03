use sqlx::PgPool;
use warp::http::StatusCode;
use serde_json::json;
use serde::{Deserialize, Serialize};
use time::{OffsetDateTime, format_description::well_known::Rfc3339};
use uuid::Uuid;

use crate::Session;
use crate::utils::{deserialize_datetime, deserialize_option_datetime, serialize_datetime, serialize_option_datetime};

#[derive(Debug, Deserialize, Serialize, sqlx::FromRow)]
pub struct Event {
    pub id: String,
    pub user_id: String,
    pub title: String,
    pub description: Option<String>,
    pub duration: i32,
    #[serde(serialize_with = "serialize_datetime", deserialize_with = "deserialize_datetime")]
    pub date: OffsetDateTime,
    pub accepted: bool,
    #[serde(serialize_with = "serialize_option_datetime", deserialize_with = "deserialize_option_datetime")]
    pub created_at: Option<OffsetDateTime>,
    #[serde(serialize_with = "serialize_option_datetime", deserialize_with = "deserialize_option_datetime")]
    pub updated_at: Option<OffsetDateTime>,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct EventInput {
    pub title: String,
    pub description: Option<String>,
    pub duration: i32,
    pub date: String,
}

impl EventInput {
    pub fn to_event(&self, user_id: String) -> Result<Event, time::Error> {
        let parsed_date = OffsetDateTime::parse(&self.date, &Rfc3339)?;
        Ok(Event {
            id: Uuid::new_v4().to_string(),
            user_id,
            title: self.title.clone(),
            description: self.description.clone(),
            duration: self.duration,
            date: parsed_date,
            accepted: false,
            created_at: Some(OffsetDateTime::now_utc()),
            updated_at: Some(OffsetDateTime::now_utc()),
        })
    }
}

// GET /events
pub async fn get_events(
    session: Option<Session>,
    pool: PgPool,
) -> Result<impl warp::Reply, warp::Rejection> {
    if session.is_some() {
        match sqlx::query_as!(
            Event,
            "SELECT * FROM events WHERE user_id = $1",
            session.unwrap().identity.id,
        )
        .fetch_all(&pool)
        .await
        {
            Ok(events) => Ok(warp::reply::with_status(
                warp::reply::json(&events),
                StatusCode::OK,
            )),
            Err(_) => Ok(warp::reply::with_status(
                warp::reply::json(&json!({"error": "Internal Server Error"})),
                StatusCode::INTERNAL_SERVER_ERROR,
            )),
        }
    } else {
        let error_message = json!({ "error": "Unauthorized" });
        Ok(warp::reply::with_status(
            warp::reply::json(&error_message),
            StatusCode::UNAUTHORIZED,
        ))
    }
}

// POST /events
pub async fn post_events(
    event_input: EventInput,
    session: Option<Session>,
    pool: PgPool,
) -> Result<impl warp::Reply, warp::Rejection> {
    if let Some(session) = session {
        match event_input.to_event(session.identity.id.clone()) {
            Ok(event) => {
                match sqlx::query!(
                    r#"
                    INSERT INTO events (id, user_id, title, description, duration, date, accepted, created_at, updated_at)
                    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
                    "#,
                    Uuid::new_v4().to_string(),
                    session.identity.id,
                    event.title,
                    event.description,
                    event.duration,
                    event.date,
                    false,
                    OffsetDateTime::now_utc(),
                    OffsetDateTime::now_utc(),
                )
                .execute(&pool)
                .await
                {
                    Ok(_) => Ok(warp::reply::with_status(
                        warp::reply::json(&json!(event)),
                        StatusCode::CREATED,
                    )),
                    Err(_) => Ok(warp::reply::with_status(
                        warp::reply::json(&json!({"error": "Internal Server Error"})),
                        StatusCode::INTERNAL_SERVER_ERROR,
                    )),
                }
            }
            Err(_) => Ok(warp::reply::with_status(
                warp::reply::json(&json!({"error": "Invalid date format"})),
                StatusCode::BAD_REQUEST,
            )),
        }
    } else {
        let error_message = json!({ "error": "Unauthorized" });
        Ok(warp::reply::with_status(
            warp::reply::json(&error_message),
            StatusCode::UNAUTHORIZED,
        ))
    }
}
