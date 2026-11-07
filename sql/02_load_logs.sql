INSERT INTO search.search_logs (timestamp, user_id, search_query)
SELECT timestamp, user_id, search_query
FROM file('data/logs.parquet', Parquet)
WHERE timestamp IS NOT NULL
  AND user_id IS NOT NULL
  AND search_query IS NOT NULL;
