use sqlx::postgres::PgPoolOptions;
use sqlx::Pool;

pub async fn create_pool(database_url: &str) -> Pool<sqlx::Postgres> {
    PgPoolOptions::new()
        .max_connections(5)
        .connect(database_url)
        .await
        .expect("Failed to create pool")
}
