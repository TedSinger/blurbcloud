-- name: init
create table if not exists blurb (
    id text,
    version int,
    body text,
    primary key (id, version)
);

-- name: get_blurb
select id, version, body from blurb
    where id = :id
    and version = (select max(version) from blurb where id = :id)
    union all select :id as "id", 0 as "version",
    '<u><em style="background-color:: rgb(255, 240, 201)">Blurb.cloud</em></u>
     is a shared, local billboard. Anyone who sees a blurb can change the blurb.' as "body";

-- name: put_blurb
insert into blurb (id, version, body)
    select :id, :version, :body
    where :version > (select coalesce(max(version), 0)
        from blurb
        where id = :id);
