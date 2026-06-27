-- name: CreateTessts :one
insert into testss (id, name) values ($1, $2) returning id, name

-- name: GETTestss :many
select * from testss where id = $1