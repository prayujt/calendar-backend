use warp::Filter;
use warp::http::StatusCode;

use serde_json::json;

use ory_kratos_client::apis::configuration::Configuration;
use ory_kratos_client::apis::frontend_api::to_session;
use ory_kratos_client::models::Session;


#[tokio::main]
async fn main() {
    dotenv::dotenv().ok();

    let kratos_configuration = Configuration {
        base_path: "https://idp.prayujt.com".to_owned(),
        ..Default::default()
    };

    let with_session = warp::cookie::optional("ory_kratos_session")
        .and(warp::any().map(move || kratos_configuration.clone()))
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
    kratos_configuration: Configuration,
) -> Result<Option<Session>, warp::Rejection> {
    let token = session_cookie.as_deref();
    println!("Session Cookie: {:?}", token);

    let result = to_session(
        &kratos_configuration,
        token,
        token,
        None,
    ).await;
    println!("Session: {:?}", result);

    match result {
        Ok(session) => Ok(Some(session)),
        Err(_) => Ok(None),
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
