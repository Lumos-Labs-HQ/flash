mod flash_gen;

use flash_gen::db::Queries;
use sqlx::PgPool;

#[tokio::main]
async fn main() {
    dotenvy::dotenv().ok();
    let db_url = std::env::var("DATABASE_URL").expect("DATABASE_URL not set in .env");

    let pool = PgPool::connect(&db_url)
        .await
        .expect("Failed to connect to database");

    let queries = Queries::new(pool);

    // Test :one (full model return)
    let user = queries.get_user(1).await;
    println!("get_user: {:?}", user.unwrap().bio);

    // Test :one (scalar)
    let count = queries.count_users().await;
    println!("count_users: {:?}", count);

    // Test :many
    let users = queries.list_active_users().await;
    println!("list_active_users: {:?}", users);

    // Test :one with boolean scalar
    let exists = queries.user_exists(1).await;
    println!("user_exists: {:?}", exists);
}
