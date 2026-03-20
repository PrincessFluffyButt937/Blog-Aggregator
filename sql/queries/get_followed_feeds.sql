-- name: GetFollows :many
SELECT feed_follows.*, users.name AS user_name, feeds.name AS feed_name FROM feed_follows
INNER JOIN users ON users.id = feed_follows.user_id
INNER JOIN feeds ON feeds.id = feed_follows.feed_id
WHERE feed_follows.user_id = $1;
