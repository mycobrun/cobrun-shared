-- Rollback: 006_create_ratings

DROP TABLE IF EXISTS user_rating_summaries;
DROP TABLE IF EXISTS rating_responses;

DROP INDEX IF EXISTS idx_ratings_created_at;
DROP INDEX IF EXISTS idx_ratings_overall_score;
DROP INDEX IF EXISTS idx_ratings_ratee_id;
DROP INDEX IF EXISTS idx_ratings_rater_id;
DROP INDEX IF EXISTS idx_ratings_trip_id;
DROP TABLE IF EXISTS ratings;





