SELECT '1. нет строк, где query = tag' AS name, count() = 0 AS passed
FROM search.query_tags WHERE query = tag;

SELECT '2. нет строк, где users < 5' AS name, count() = 0 AS passed
FROM search.query_tags WHERE users < 5;

SELECT '3. не больше 10 тегов на запрос' AS name, count() = 0 AS passed
FROM (SELECT query FROM search.query_tags GROUP BY query HAVING count() > 10);

SELECT '4. rank идёт 1..N без пропусков' AS name, count() = 0 AS passed
FROM (SELECT query FROM search.query_tags GROUP BY query HAVING min(rank) != 1 OR max(rank) != count());

SELECT '5. query и tag нормализованы' AS name, count() = 0 AS passed
FROM search.query_tags
WHERE query != trimBoth(replaceRegexpAll(lowerUTF8(query), '\\s+', ' '))
   OR tag   != trimBoth(replaceRegexpAll(lowerUTF8(tag),   '\\s+', ' '));

SELECT '6. «Платье » и «платье» дают одинаковые теги' AS name,
       length(a) > 0 AND a = b AS passed
FROM
(
    SELECT groupArray((tag, users)) AS a
    FROM (SELECT tag, users FROM search.query_tags
          WHERE query = trimBoth(replaceRegexpAll(lowerUTF8('Платье '), '\\s+', ' ')) ORDER BY rank)
) AS x,
(
    SELECT groupArray((tag, users)) AS b
    FROM (SELECT tag, users FROM search.query_tags
          WHERE query = trimBoth(replaceRegexpAll(lowerUTF8('платье'), '\\s+', ' ')) ORDER BY rank)
) AS y;
