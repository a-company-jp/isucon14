ALTER TABLE chairs
    ADD COLUMN total_distance INT NOT NULL DEFAULT 0 COMMENT '総移動距離',
    ADD COLUMN total_distance_updated_at DATETIME(6) NULL COMMENT '総移動距離の更新日';