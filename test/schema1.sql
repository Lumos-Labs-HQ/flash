create table posts (
    id serial primary key,
    user_id integer references users(id) on delete cascade,
    title varchar(255) not null,
    content text not null,
    created_at timestamp with time zone default current_timestamp
);
create table users (
    id serial primary key,
    username varchar(100) not null,
    email varchar(100) not null,
    created_at timestamp with time zone default current_timestamp
);
create table comments (
    id serial primary key,
    post_id integer references posts(id) on delete cascade,
    user_id integer references users(id) on delete cascade,
    content text not null,
    created_at timestamp with time zone default current_timestamp
);