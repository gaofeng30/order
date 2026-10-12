UPDATE action_audits AS a SET
  entry_kind='LEGACY_EVIDENCE',actor_kind='MERCHANT',
  actor_scope_hash=sha256(convert_to(concat('LEGACY:',COALESCE(a.actor_account_id_snapshot::text,'NONE'),':',a.actor_user_id::text),'UTF8')),
  actor_user_id=CASE WHEN EXISTS(SELECT 1 FROM miniprogram_users AS u WHERE u.id=a.actor_user_id) THEN a.actor_user_id ELSE NULL END,
  operation_key_hash=NULL,target_key_hash=NULL,request_id_hash=sha256(a.request_id),
  before_state_json=CASE WHEN a.state_before IS NULL THEN NULL ELSE jsonb_build_object('legacy_state',a.state_before) END,
  after_state_json=CASE WHEN a.state_after IS NULL THEN NULL ELSE jsonb_build_object('legacy_state',a.state_after) END,
  response_json=NULL;
