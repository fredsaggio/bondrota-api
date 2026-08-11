-- Limpeza diaria dos dados operacionais fora da janela de retencao.
--
-- Pre-requisito: o planning_cron.sql ja precisa ter sido executado, porque este
-- script reutiliza o schema bondrota_internal e o segredo
-- bondrota_planning_cron_secret (o endpoint /internal e protegido pelo mesmo
-- PLANNING_CRON_SECRET).
--
-- Antes de rodar, substitua <RENDER_RETENTION_ENDPOINT> pela URL completa
-- https://SEU-SERVICO.onrender.com/api/v1/internal/retencao/limpar

select vault.create_secret(
  '<RENDER_RETENTION_ENDPOINT>',
  'bondrota_retention_endpoint',
  'BondRota internal retention cleanup endpoint'
);

create or replace function bondrota_internal.invoke_retention_cleanup()
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
  where name = 'bondrota_retention_endpoint';

  select decrypted_secret
  into bearer_secret
  from vault.decrypted_secrets
  where name = 'bondrota_planning_cron_secret';

  if endpoint_url is null or bearer_secret is null then
    raise exception 'BondRota retention cron secrets are not configured';
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

revoke all on function bondrota_internal.invoke_retention_cleanup() from public;

-- 04:10 UTC = 01:10 em America/Maceio: fora do horario de operacao, quando o
-- lock nas tabelas nao concorre com reservas e viagens em andamento.
select cron.schedule(
  'bondrota-retention-cleanup',
  '10 4 * * *',
  $$select bondrota_internal.invoke_retention_cleanup();$$
);
