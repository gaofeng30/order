import json,pathlib,subprocess,sys,tempfile,unittest
from unittest.mock import patch
SOURCE=(pathlib.Path(__file__).with_name('ui2.py')).read_text().split("if group=='customer':")[0]
class RunnerEvidence(unittest.TestCase):
 def setUp(self):
  self.tmp=tempfile.TemporaryDirectory();self.addCleanup(self.tmp.cleanup);self.root=pathlib.Path(self.tmp.name);self.fixture=self.root/'fixture';self.fixture.mkdir();(self.fixture/'utils').mkdir();(self.fixture/'fixture-manifest.json').write_text(json.dumps({'warning':'Not an uploadable candidate; no native payment evidence'}));(self.fixture/'utils/apiClient.js').write_text('Synthetic fixture: all network APIs blocked')
 def load(self,out):
  scope={}
  with patch.object(sys,'argv',['ui2.py',str(self.fixture),'unused',str(out)]):exec(compile(SOURCE,'ui2.py','exec'),scope)
  return scope
 def test_refuse_existing_result_directory(self):
  out=self.root/'old';out.mkdir();(out/'result.json').write_text('{"result":"PASS"}')
  with self.assertRaises(FileExistsError):self.load(out)
 def test_non_json_keeps_raw_failure(self):
  out=self.root/'new';s=self.load(out)
  with patch.object(subprocess,'run',return_value=subprocess.CompletedProcess([],2,'not json','diagnostic error')):
   with self.assertRaises(Exception):s['call']('synthetic_tool')
  r=json.loads((out/'calls.json').read_text())[0];self.assertEqual(r['stdout'],'not json');self.assertEqual(r['stderr'],'diagnostic error');self.assertEqual(r['returncode'],2)
 def test_timeout_keeps_partial_raw(self):
  out=self.root/'timeout';s=self.load(out)
  with patch.object(subprocess,'run',side_effect=subprocess.TimeoutExpired('synthetic_tool',45,output=b'partial',stderr=b'timeout detail')):
   with self.assertRaises(subprocess.TimeoutExpired):s['call']('synthetic_tool')
  r=json.loads((out/'calls.json').read_text())[0];self.assertTrue(r['timed_out']);self.assertEqual(r['stdout'],'partial');self.assertEqual(r['stderr'],'timeout detail')
 def test_nonzero_exit_is_failure_even_if_json_claims_ok(self):
  out=self.root/'nonzero';s=self.load(out)
  with patch.object(subprocess,'run',return_value=subprocess.CompletedProcess([],1,'{"ok":true,"result":{}}','error')):
   with self.assertRaises(RuntimeError):s['call']('synthetic_tool')
 def test_real_invalid_utf8_bytes_are_preserved(self):
  out=self.root/'bytes';s=self.load(out);original_run=subprocess.run
  def invalid_process(args,**kwargs):return original_run([sys.executable,'-c',"import sys;sys.stdout.buffer.write(bytes([255]));sys.stderr.buffer.write(bytes([254]));sys.exit(2)"],**kwargs)
  with patch.object(subprocess,'run',side_effect=invalid_process):
   with self.assertRaises(Exception):s['call']('synthetic_tool')
  r=json.loads((out/'calls.json').read_text())[0];self.assertEqual(r['stdout_base64'],'/w==');self.assertEqual(r['stderr_base64'],'/g==');self.assertEqual(r['returncode'],2)
if __name__=='__main__':unittest.main()
