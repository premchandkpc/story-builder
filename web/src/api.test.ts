import { afterEach, describe, expect, it, vi } from 'vitest';
import { api } from './api';
afterEach(() => vi.unstubAllGlobals());
function mockResponse(response: Response) { vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response)); }
describe('draft lifecycle and streaming', () => {
 it('rejects a draft with an empty 204 response', async () => {
  mockResponse(new Response(null, {status:204}));
  await expect(api.rejectDraft('draft')).resolves.toBeUndefined();
 });
 it('surfaces API errors', async () => {
  mockResponse(new Response(JSON.stringify({message:'Canon changed'}),{status:409}));
  await expect(api.acceptDraft('draft')).rejects.toThrow('Canon changed');
 });
 it('reassembles fragmented UTF-8 and long SSE payloads without losing prose', async () => {
  const prose='மீரா 🌙 她选择回家。\n'.repeat(4000);
  const draft={id:'draft',status:'READY',prose};
  const payload=`event: chunk\ndata: ${JSON.stringify({text:prose})}\n\nevent: complete\ndata: ${JSON.stringify(draft)}\n\n`;
  const bytes=new TextEncoder().encode(payload);
  mockResponse(new Response(new ReadableStream({start(controller){
   for(let offset=0;offset<bytes.length;offset+=37)controller.enqueue(bytes.slice(offset,offset+37));
   controller.close();
  }})));
  let text='';
  await expect(api.generate('scene',chunk=>{text+=chunk})).resolves.toEqual(draft);
  expect(text).toBe(prose);
 });
 it('surfaces streamed generation errors', async () => {
  mockResponse(new Response('event: error\ndata: {"message":"accepted scenes are locked"}\n\n'));
  await expect(api.generate('scene',()=>{})).rejects.toThrow('accepted scenes are locked');
 });
 it('rejects an interrupted stream without a completed draft', async () => {
  mockResponse(new Response('event: chunk\ndata: {"text":"partial"}\n\n'));
  await expect(api.generate('scene',()=>{})).rejects.toThrow('without a draft');
 });
});
