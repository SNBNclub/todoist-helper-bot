create table if not exists chats (
    id BIGINT PRIMARY KEY,
    name varchar(100) NOT NULL
);

create table if not exists todoist_users (
    id VARCHAR(100) PRIMARY KEY,
    name VARCHAR(100) NOT NULL
);

create table if not exists stat (
    chat_id BIGINT NOT NULL UNIQUE,
    time_count BIGINT NOT NULL DEFAULT 0,
    FOREIGN KEY (chat_id) REFERENCES chats(id)
);

-- TODO :: add date of tasks
-- TODO :: store todoist task id as primary
create table if not exists tasks (
    chat_id BIGINT NOT NULL,
    content varchar(1000) not null,
    time_spent INT not null,
    FOREIGN KEY (chat_id) REFERENCES chats(id)
);

create table if not exists chat_to_todoist (
    chat_id BIGINT NOT NUll,
    todoist_id VARCHAR(100) NOT NULL,
    FOREIGN KEY (chat_id) REFERENCES chats(id),
    FOREIGN KEY (todoist_id) REFERENCES todoist_users(id)
);

CREATE PROCEDURE RecordStats(IN chatID BIGINT, IN content VARCHAR(1000), IN timeSpent INT)
LANGUAGE plpgsql
AS $$
BEGIN
    PERFORM time_count FROM stat WHERE chat_id = chatID FOR UPDATE;
    INSERT INTO tasks (chat_id, content, time_spent) VALUES (chatID, content, timeSpent);
    INSERT INTO stat (chat_id) VALUES (chatID) ON CONFLICT (chat_id) DO UPDATE SET time_count = stat.time_count + timeSpent;
END;
$$;