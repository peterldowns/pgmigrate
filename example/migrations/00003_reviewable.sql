CREATE VIEW reviewable_blobs AS
SELECT 
  *
FROM blobs
WHERE status = 'pending_review';
  
