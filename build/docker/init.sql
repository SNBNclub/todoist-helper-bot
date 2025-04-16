create table if not exists chats (
    id BIGINT PRIMARY KEY,
    name varchar(100) NOT NULL
);

# TODO :: or bigint id
create table if not exists todoist_users (
    id VARCHAR(100) PRIMARY KEY,
    name VARCHAR(100) NOT NULL
);

create table if not exists stat (
    chat_id BIGINT NOT NULL,
    time_count BIGINT,
    FOREIGN KEY (chat_id) REFERENCES tg_users(id)
);

create table if not exists chat_to_todoist (
    chat_id BIGINT NOT NUll,
    todoist_id VARCHAR(100) NOT NULL,
    FOREIGN key (chat_id) REFERENCES chats(id),
    FOREIGN KEY (todoist_id) REFERENCES todoist_users(id)
);