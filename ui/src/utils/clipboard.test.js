import assert from 'node:assert/strict';
import { test } from 'node:test';
import { copyText } from './clipboard.js';

function createDocument(copyResult = true) {
  const nodes = [];
  let selectedText;
  let focused = false;
  const document = {
    activeElement: { focus() { focused = true; } },
    body: { appendChild(node) { nodes.push(node); } },
    createElement() {
      return {
        style: {},
        setAttribute() {},
        focus() {},
        select() { selectedText = this.value; },
        remove() { nodes.splice(nodes.indexOf(this), 1); },
      };
    },
    execCommand(command) {
      assert.equal(command, 'copy');
      return copyResult;
    },
  };
  return { document, nodes, text: () => selectedText, focused: () => focused };
}

test('copyText awaits the Clipboard API', async () => {
  let copied;
  await copyText('vpn://example', {
    clipboard: { async writeText(text) { copied = text; } },
  });
  assert.equal(copied, 'vpn://example');
});

// HTTP deployments have no navigator.clipboard; both profile formats crashed.
for (const text of ['[Interface]\nPrivateKey = example\n', 'vpn://example']) {
  test(`copyText supports missing Clipboard API for ${text.startsWith('vpn:') ? 'VPN keys' : 'AWG configs'}`, async () => {
    const dom = createDocument();
    await copyText(text, {}, dom.document);
    assert.equal(dom.text(), text);
    assert.deepEqual(dom.nodes, []);
    assert.equal(dom.focused(), true);
  });
}

test('copyText falls back when the Clipboard API rejects', async () => {
  const dom = createDocument();
  await copyText('vpn://example', {
    clipboard: { async writeText() { throw new Error('Permission denied'); } },
  }, dom.document);
  assert.equal(dom.text(), 'vpn://example');
  assert.deepEqual(dom.nodes, []);
});

test('copyText reports rejected fallback and removes temporary text', async () => {
  const dom = createDocument(false);
  await assert.rejects(copyText('vpn://example', {}, dom.document));
  assert.deepEqual(dom.nodes, []);
  assert.equal(dom.focused(), true);
});

test('copyText cleans up when the fallback throws', async () => {
  const dom = createDocument();
  dom.document.execCommand = () => { throw new Error('Copy blocked'); };
  await assert.rejects(copyText('vpn://example', {}, dom.document));
  assert.deepEqual(dom.nodes, []);
  assert.equal(dom.focused(), true);
});
