use warp::Filter;
use warp::http::StatusCode;

use serde_json::json;

use reqwest::Client;

use serde::{Deserialize, Serialize};

#[derive(Debug, Deserialize, Serialize)]
struct Session {
    id: String,
    active: bool,
    identity: Identity,
}

#[derive(Debug, Deserialize, Serialize)]
struct Identity {
    id: String,
    state: String,
    traits: Traits,
}

#[derive(Debug, Deserialize, Serialize)]
#[serde(rename_all = "camelCase")]
struct Traits {
    email: String,
    first_name: String,
    last_name: String,
    username: String,
}

#[tokio::main]
async fn main() {
    dotenv::dotenv().ok();

    let kratos_base_url = "https://idp.prayujt.com";

    let client = Client::new();

    let with_session = warp::cookie::optional("ory_kratos_session")
        .and(warp::any().map(move || client.clone()))
        .and(warp::any().map(move || kratos_base_url.to_string()))
        .and_then(verify_session);

    let get_events_route = warp::get()
        .and(warp::path("events"))
        .and(with_session.clone())
        .and_then(get_events);

    let post_events_route = warp::post()
        .and(warp::path("events"))
        .and(warp::body::json())
        .and(with_session)
        .and_then(post_events);

    let routes = get_events_route.or(post_events_route);

    println!("Server running on 0.0.0.0:8080");
    warp::serve(routes).run(([0, 0, 0, 0], 8080)).await;
}

async fn verify_session(
    session_cookie: Option<String>,
    client: Client,
    kratos_base_url: String,
) -> Result<Option<Session>, warp::Rejection> {
    if let Some(token) = session_cookie {
        let url = format!("{}/sessions/whoami", kratos_base_url);
        let response = client
            .get(&url)
            .header("Cookie", format!("ory_kratos_session={:?}", token))
            .send()
            .await;

        if let Ok(response) = response {
            if response.status().is_success() {
                if let Ok(session) = response.json::<Session>().await {
                    return Ok(Some(session));
                }
            }
        }
        Ok(None)
    } else {
        Ok(None)
    }
}

// GET /events
async fn get_events(session: Option<Session>) -> Result<impl warp::Reply, warp::Rejection> {
    if let Some(session) = session {
        Ok(warp::reply::with_status(
            warp::reply::json(&session),
            StatusCode::OK
        ))
    } else {
        let error_message = json!({ "error": "Unauthorized" });
        Ok(warp::reply::with_status(
            warp::reply::json(&error_message),
            StatusCode::UNAUTHORIZED
        ))
    }
}

// POST /events
async fn post_events(_body: serde_json::Value, session: Option<Session>) -> Result<impl warp::Reply, warp::Rejection> {
    if let Some(session) = session {
        Ok(warp::reply::with_status(
            warp::reply::json(&session),
            StatusCode::OK
        ))
    } else {
        let error_message = json!({ "error": "Unauthorized" });
        Ok(warp::reply::with_status(
            warp::reply::json(&error_message),
            StatusCode::UNAUTHORIZED
        ))
    }
}
