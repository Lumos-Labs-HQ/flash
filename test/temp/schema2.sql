create table users (
 id serial primary key,
 username varchar(100) not null,
 email varchar(100) not null,
 created_at timestamp with time zone default current_timestamp
);
create table posts (
 id serial primary key,
 user_id integer references users(id) on delete cascade,
 title varchar(255) ,
 content text not null,
 created_at timestamp with time zone default current_timestamp
);
create index idx_posts_user_id on posts(user_id);