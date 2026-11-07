CREATE TABLE IF NOT EXISTS search.query_tags
(
    query String,
    tag   String,
    users UInt32,
    rank  UInt8
)
ENGINE = MergeTree
ORDER BY (query, rank);

TRUNCATE TABLE search.query_tags;

INSERT INTO search.query_tags (query, tag, users, rank)
WITH
    cleaned AS
    (
        SELECT
            user_id,
            timestamp AS ts,
            trimBoth(replaceRegexpAll(lowerUTF8(search_query), '\\s+', ' ')) AS q
        FROM search.search_logs
        WHERE q != ''
    ),
    pairs AS
    (
        SELECT
            user_id,
            lagInFrame(q)  OVER w AS query,
            q                     AS tag,
            lagInFrame(ts) OVER w AS prev_ts,
            ts
        FROM cleaned
        WINDOW w AS (
            PARTITION BY user_id
            ORDER BY ts, q
            ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING
        )
    ),
    counted AS
    (
        SELECT query, tag, uniqExact(user_id) AS users
        FROM pairs
        WHERE query != ''
          AND query != tag
          AND toUnixTimestamp64Milli(ts) - toUnixTimestamp64Milli(prev_ts) <= 1800000
        GROUP BY query, tag
        HAVING users >= 5
    ),
    ranked AS
    (
        SELECT query, tag, users,
               row_number() OVER (PARTITION BY query ORDER BY users DESC, tag ASC) AS rank
        FROM counted
    )
SELECT query, tag, users, rank
FROM ranked
WHERE rank <= 10;

