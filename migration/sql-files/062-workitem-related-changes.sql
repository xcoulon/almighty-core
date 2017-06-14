-- add a column to record the timestamp of the latest addition/change/removal of a comment on a workitem
ALTER TABLE work_items ADD COLUMN commented_at timestamp with time zone;
-- add a column to record the timestamp of the latest addition/change/removal of a link to/from a workitem
ALTER TABLE work_items ADD COLUMN linked_at timestamp with time zone;

-- trigger to fill the `commented_at` column when a comment is added or removed (soft delete, it's a record update)
CREATE FUNCTION workitem_comment_insert_timestamp() RETURNS trigger AS $$
    BEGIN
        UPDATE work_items wi SET commented_at = NEW.created_at WHERE wi.id::text = NEW.parent_id;
        RETURN NEW;
    END;
$$ LANGUAGE plpgsql;

CREATE FUNCTION workitem_comment_softdelete_timestamp() RETURNS trigger AS $$
    BEGIN
        UPDATE work_items wi SET commented_at = NEW.deleted_at WHERE wi.id::text = NEW.parent_id;
        RETURN NEW;
    END;
$$ LANGUAGE plpgsql;


CREATE TRIGGER workitem_comment_insert_trigger AFTER INSERT ON comments 
    FOR EACH ROW
    EXECUTE PROCEDURE workitem_comment_insert_timestamp();
CREATE TRIGGER workitem_comment_softdelete_trigger AFTER UPDATE OF deleted_at ON comments 
    FOR EACH ROW
    EXECUTE PROCEDURE workitem_comment_softdelete_timestamp();
