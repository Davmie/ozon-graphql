drop table if exists posts cascade;
create table posts (
    id uuid primary key,
    title text not null,
    content text not null,
    commentsEnabled boolean not null
);

drop table if exists comments cascade;
create table comments (
    id uuid primary key,
    postId uuid not null,
    parentId uuid,
    content VARCHAR(2000) not null,
    createdAt timestamp not null,
    foreign key (postId) references posts (id),
    foreign key (parentId) references comments (id)
);

create or replace function notify_new_comment() returns trigger as $$
BEGIN
    perform pg_notify('new_comment', new.id::text || ',' || new.postId::text);
    return NULL;
END;
$$ language plpgsql;

create trigger new_comment_trigger
    after insert on comments
    for each row
    execute procedure notify_new_comment();