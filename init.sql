-- 后台任务队列配置表
CREATE TABLE public.queue_configs
(
    id                 bigserial                                   NOT NULL,
    queue_name         varchar(100)                                NOT NULL,
    task_table         varchar(100) DEFAULT '':: character varying NOT NULL,
    max_workers        int4         DEFAULT 1                      NOT NULL,
    frequency_interval int4         default 3                      not null,
    priority           int4         DEFAULT 1                      NOT NULL,
    server_id          varchar(50)                                 NULL,
    status             int4         DEFAULT 1                      NOT NULL,
    created_at         timestamptz                                 NULL DEFAULT NOW(),
    updated_at         timestamptz                                 NULL DEFAULT NOW(),
    is_delete          int2         DEFAULT 0                      NOT NULL,
    CONSTRAINT queue_configs_pkey PRIMARY KEY (id),
    CONSTRAINT queue_configs_queue_name_key UNIQUE (queue_name)
);
CREATE INDEX idx_queue_configs_server_id ON public.queue_configs USING btree (server_id);
CREATE INDEX idx_queue_configs_status ON public.queue_configs USING btree (status);
CREATE UNIQUE INDEX uni_queue_configs_queue_name ON public.queue_configs USING btree (queue_name);

---------------------------------------------------------------------------------------------------

-- 媒体路由
CREATE TABLE public.media_hosts
(
    id         bigserial                                       NOT NULL,
    media_type int4          DEFAULT 0                         NOT NULL,
    url_code   varchar(200)  DEFAULT '':: character varying    NOT NULL,
    url_name   varchar(100)  DEFAULT '':: character varying    NOT NULL,
    url_host   varchar(100)  DEFAULT '':: character varying    NOT NULL,
    url_scheme varchar(10)   DEFAULT '':: character varying    NOT NULL,
    url_path   varchar(200)  DEFAULT '':: character varying    NOT NULL,
    url_query  varchar(1000) DEFAULT '':: character varying    NOT NULL,
    req_method varchar(10)   DEFAULT 'GET':: character varying NOT NULL,
    state      int4          DEFAULT 1                         NOT NULL,
    created_at timestamptz                                     NULL DEFAULT NOW(),
    updated_at timestamptz                                     NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT media_hosts_pkey PRIMARY KEY (id)
);
CREATE INDEX idx_media_hosts_url_code ON public.media_hosts USING btree (media_type, url_code);

---------------------------------------------------------------------------------------------------

-- 任务表建表语句
CREATE TABLE public.task_email
(
    id            int8 GENERATED ALWAYS AS IDENTITY                                       NOT NULL,
    batch_no      varchar(80)                                                             NOT NULL,
    queue_name    varchar(80)                                                             NOT NULL,
    task_params   jsonb                                                                   NOT NULL DEFAULT '{}',
    state         varchar(20)                                                             NOT NULL,
    schedule_time timestamp   DEFAULT '2000-01-01 00:00:00':: timestamp without time zone NOT NULL,
    start_time    timestamp   DEFAULT '2000-01-01 00:00:00':: timestamp without time zone NOT NULL,
    end_time      timestamp   DEFAULT '2000-01-01 00:00:00':: timestamp without time zone NOT NULL,
    cost_time     int4        DEFAULT 0                                                   NOT NULL,
    retry_count   int4        DEFAULT 0                                                   NOT NULL,
    max_retries   int4        DEFAULT 0                                                   NOT NULL,
    exec_result   jsonb                                                                   NOT NULL DEFAULT '{}',
    server_id     varchar(50) DEFAULT '':: character varying                              NOT NULL,
    task_type     varchar(30) DEFAULT 'default'                                           NOT NULL,
    created_at    timestamp                                                               NULL DEFAULT NOW(),
    updated_at    timestamp                                                               NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT task_email_pkey PRIMARY KEY (id)
);
CREATE INDEX idx_schedule_task ON public.task_email USING btree (schedule_time, state, server_id, queue_name);
CREATE INDEX idx_created_at ON public.task_email USING btree (created_at);
CREATE INDEX idx_batch_no ON public.task_email USING btree (batch_no);
CREATE INDEX idx_queue_name ON public.task_email USING btree (queue_name);