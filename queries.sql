-- name: init
create table if not exists blurb (
    id text,
    version int,
    body text,
    primary key (id, version)
);

-- name: get_blurb
select id, version, body from blurb where id = :id;

-- name: put_blurb
insert into blurb (id, version, body) values (:id, :version, :body) returning version;
