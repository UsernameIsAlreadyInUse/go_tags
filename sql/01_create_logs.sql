CREATE DATABASE IF NOT EXISTS search;

CREATE TABLE IF NOT EXISTS search.search_logs
(
    timestamp    DateTime64(3, 'UTC'),
    user_id      UInt64,
    search_query String
)
ENGINE = MergeTree
ORDER BY (user_id, timestamp);
