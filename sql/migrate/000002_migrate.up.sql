CREATE INDEX idx_users_access_token ON users(access_token);
CREATE INDEX idx_chairs_is_active ON chairs(is_active);
CREATE INDEX idx_chairs_owner_id ON chairs(owner_id);
CREATE INDEX idx_ride_statuses_ride_id_created_at ON ride_statuses(ride_id, created_at DESC);
CREATE INDEX idx_rides_user_id ON rides(user_id);
CREATE INDEX idx_rides_chair_id ON rides(chair_id);
CREATE INDEX idx_chair_locations_chair_id_created_at ON chair_locations(chair_id, created_at DESC);