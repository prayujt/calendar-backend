use warp::Filter;
use reqwest::Client;
use dotenv::dotenv;
use std::env;

mod events;
mod db;

#[derive(Debug, serde::Deserialize, serde::Serialize)]
struct Session {
    id: String,
    active: bool,
    identity: Identity,
}

#[derive(Debug, serde::Deserialize, serde::Serialize)]
struct Identity {
    id: String,
    state: String,
    traits: Traits,
}

#[derive(Debug, serde::Deserialize, serde::Serialize)]
#[serde(rename_all = "camelCase")]
struct Traits {
    email: String,
    first_name: String,
    last_name: String,
    username: String,
}

#[tokio::main]
async fn main() {
    dotenv().ok();

    let kratos_base_url = env::var("KRATOS_BASE_URL").unwrap_or_else(|_| "https://idp.prayujt.com".to_string());
    let database_url = env::var("DATABASE_URL").expect("DATABASE_URL must be set");
    println!("Using Kratos base URL: {}", kratos_base_url);
    println!("Using database URL: {}", database_url);

    let client = Client::new();
    let pool = db::create_pool(&database_url).await;

    let with_session = warp::cookie::optional("ory_kratos_session")
        .and(warp::any().map(move || client.clone()))
        .and(warp::any().map(move || kratos_base_url.clone()))
        .and_then(verify_session);
    let with_pool = warp::any().map(move || pool.clone());

    let get_events_route = warp::get()
        .and(warp::path("events"))
        .and(with_session.clone())
        .and(with_pool.clone())
        .and_then(events::get_events);

    let post_events_route = warp::post()
        .and(warp::path("events"))
        .and(warp::body::json())
        .and(with_session)
        .and(with_pool)
        .and_then(events::post_events);

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
