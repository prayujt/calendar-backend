use sqlx::PgPool;
use warp::http::StatusCode;
use serde_json::json;
use serde::{Deserialize, Serialize};
use chrono::{DateTime, Utc};

use crate::Session;

#[derive(Debug, Deserialize, Serialize)]
pub struct Event {
    pub id: String,
    pub user_id: String,
    pub title: String,
    pub description: String,
    pub duration: i32,
    #[serde(with = "chrono::serde::ts_seconds")]
    pub date: DateTime<Utc>,
    pub accepted: bool,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
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
    event: Event,
    session: Option<Session>,
    pool: PgPool,
) -> Result<impl warp::Reply, warp::Rejection> {
    if session.is_some() {
        match sqlx::query!(
            "INSERT INTO events (title, description, duration) VALUES ($1, $2, $3)",
            event.title,
            event.description,
            event.duration,
        )
        .execute(&pool)
        .await
        {
            Ok(_) => Ok(warp::reply::with_status(
                warp::reply::json(&json!({"success": "Event created"})),
                StatusCode::CREATED,
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
