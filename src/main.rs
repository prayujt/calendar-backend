use warp::Filter;

#[tokio::main]
async fn main() {
    dotenv::dotenv().ok();

    let get_events_route = warp::get()
        .and(warp::path("events"))
        .and_then(get_events);

    let post_events_route = warp::post()
        .and(warp::path("events"))
        .and(warp::body::json())
        .and_then(post_events);

    let routes = get_events_route.or(post_events_route);

    println!("Server running on 0.0.0.0:8080");
    warp::serve(routes).run(([0, 0, 0, 0], 8080)).await;
}

// GET /events
async fn get_events() -> Result<impl warp::Reply, warp::Rejection> {
    Ok(warp::reply::json(&"Hello World!"))
}

// POST /events
async fn post_events(body: serde_json::Value) -> Result<impl warp::Reply, warp::Rejection> {
    Ok(warp::reply::json(&body))
}
