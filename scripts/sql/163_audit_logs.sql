--
-- CREATE SEQUENCE IF NOT EXISTS id_audit_logs;
--
-- -- Table Definition
-- CREATE TABLE  "public"."plugin_stage_mapping"
-- (
--     "id"             int4 AUTOINCREMENT NOT NULL,
--     "updated_on"     timestamptz,
--     "updated_by"     int4,
--     "action"         varchar(10) NOT NULL,
--
--     PRIMARY KEY ("id")
-- );