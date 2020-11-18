create table notebook.note
(
  id         int          not null auto_increment,
  title      varchar(255) not null,
  author     varchar(32)  not null,
  content    text         not null,
  plain_text text         not null,
  words      int          not null default 0,
  private    tinyint      not null default 0,
  created_at timestamp    not null default current_timestamp,
  update_at  timestamp    not null default current_timestamp on update current_timestamp,
  primary key (id),
  index update_at (update_at)
) ENGINE = InnoDB
  AUTO_INCREMENT = 1
  DEFAULT CHARSET = utf8;

create table notebook.tag
(
  id         int          not null auto_increment,
  name       varchar(255) not null,
  created_at timestamp    not null default current_timestamp,
  update_at  timestamp    not null default current_timestamp on update current_timestamp,
  primary key (id),
  unique key (name)
) ENGINE = InnoDB
  AUTO_INCREMENT = 1
  DEFAULT CHARSET = utf8;

create table notebook.note_tag
(
  id         int          not null auto_increment,
  note_id    int          not null,
  tag_id     int          not null,
  tag_name   varchar(255) not null,
  created_at timestamp    not null default current_timestamp,
  update_at  timestamp    not null default current_timestamp on update current_timestamp,
  primary key (id),
  unique key note_tag_id (note_id, tag_id),
  index tag_id (tag_id)
) ENGINE = InnoDB
  AUTO_INCREMENT = 1
  DEFAULT CHARSET = utf8;

