select vault.create_secret(
  '<RENDER_PLANNING_ENDPOINT>',
  'bondrota_planning_endpoint',
  'BondRota internal planning processor endpoint'
);

select vault.create_secret(
  '<PLANNING_CRON_SECRET>',
  'bondrota_planning_cron_secret',
  'Bearer secret for the BondRota planning cron'
);

create schema if not exists bondrota_internal;
revoke all on schema bondrota_internal from public;

create or replace function bondrota_internal.invoke_planning_processor()
returns bigint
language plpgsql
security definer
set search_path = ''
as $function$
declare
  endpoint_url text;
  bearer_secret text;
begin
  select decrypted_secret
  into endpoint_url
  from vault.decrypted_secrets
  where name = 'bondrota_planning_endpoint';

  select decrypted_secret
  into bearer_secret
  from vault.decrypted_secrets
  where name = 'bondrota_planning_cron_secret';

  if endpoint_url is null or bearer_secret is null then
    raise exception 'BondRota planning cron secrets are not configured';
  end if;

  return net.http_post(
    url := endpoint_url,
    headers := jsonb_build_object(
      'Authorization', 'Bearer ' || bearer_secret,
      'Content-Type', 'application/json'
    ),
    body := '{}'::jsonb,
    timeout_milliseconds := 55000
  );
end;
$function$;

revoke all on function bondrota_internal.invoke_planning_processor() from public;

select cron.schedule(
  'bondrota-process-planejamentos',
  '* * * * *',
  $$select bondrota_internal.invoke_planning_processor();$$
);
