insert into spaces (id, name) values ('11111111-6262-0000-0000-000000000000', 'test');
-- insert into iterations (id, name, path, space_id) values ('11111111-6262-0000-0000-000000000000', 'test area', '', '11111111-6262-0000-0000-000000000000');
insert into work_item_types (id, name, space_id) values ('11111111-6262-0000-0000-000000000000', 'Test WIT','11111111-6262-0000-0000-000000000000');
insert into work_items (id, space_id, type, fields) values (62001, '11111111-6262-0000-0000-000000000000', '11111111-6262-0000-0000-000000000000', '{"system.title":"Work item 1"}'::json);
insert into work_items (id, space_id, type, fields) values (62002, '11111111-6262-0000-0000-000000000000', '11111111-6262-0000-0000-000000000000', '{"system.title":"Work item 2"}'::json);
-- remove previous comments
delete from comments;
-- add comments linked to work items above
insert into comments (id, parent_id, body, created_at) values ( '11111111-6262-0001-0000-000000000000', '62001', 'a comment', '2017-06-13 09:00:00.0000+00');
insert into comments (id, parent_id, body, created_at) values ( '11111111-6262-0002-0000-000000000000', '62002', 'a comment', '2017-06-13 09:10:00.0000+00');
-- mark the last comment as (soft) deleted 
update comments set deleted_at = '2017-06-13 09:15:00.0000+00' where id =  '11111111-6262-0002-0000-000000000000';
