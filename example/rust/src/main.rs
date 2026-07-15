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
    
    // let parmas = CreatePostParams{
    //     user_id: 1,
    //     title: "hello".into(),
    //     body: "world".into(),
    // };
    // 
    // let userpas = CreateUserParams{
    //     name: "Hello".into(),
    //     email: "hello@example.com".into(),
    //     age: 0,
    // };
    // let post_id = Uuid::parse_str("f99e3232-7cbd-46b6-9ebe-d29118c90529").unwrap();
    let user = queries.get_user_by_email("hello@example.com").await.unwrap();
    
    println!("{:?}", user);
}
