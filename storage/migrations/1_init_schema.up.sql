CREATE TABLE IF NOT EXISTS chats (
    id BIGINT PRIMARY KEY,
    name VARCHAR(100) NOT NULL
);

CREATE TABLE IF NOT EXISTS todoist_users (
    id VARCHAR(100) PRIMARY KEY,
    name VARCHAR(100) NOT NULL
);

CREATE TABLE IF NOT EXISTS stat (
    chat_id BIGINT NOT NULL UNIQUE,
    time_count BIGINT NOT NULL DEFAULT 0,
    FOREIGN KEY (chat_id) REFERENCES chats(id)
);

CREATE TABLE IF NOT EXISTS tasks (
    chat_id BIGINT NOT NULL,
    content VARCHAR(1000) NOT NULL,
    time_spent INT NOT NULL,
    FOREIGN KEY (chat_id) REFERENCES chats(id)
);

CREATE TABLE IF NOT EXISTS chat_to_todoist (
    chat_id BIGINT NOT NULL,
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

-- // TODO :: create indexes