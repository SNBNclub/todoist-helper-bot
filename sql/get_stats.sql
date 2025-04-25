SELECT s.time_count, t.content, t.time_spent
FROM stat s
JOIN tasks t ON s.chat_id = t.chat_id
WHERE s.chat_id = $1;