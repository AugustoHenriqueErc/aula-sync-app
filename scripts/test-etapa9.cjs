// Teste de integração no Chrome real; usa apenas módulos nativos do Node 22.
// Executar da raiz: node scripts/test-etapa9.cjs
const { spawn, execFileSync } = require("node:child_process");
const fs = require("node:fs");
const path = require("node:path");
const net = require("node:net");
const assert = require("node:assert/strict");
const root = path.resolve(__dirname, "..");
const dir = fs.mkdtempSync("/tmp/aula-sync-etapa9-");
const children = [];
const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));
let ws;
async function port() {
  const server = net.createServer();
  await new Promise((resolve, reject) => {
    server.once("error", reject);
    server.listen(0, "127.0.0.1", resolve);
  });
  const value = server.address().port;
  await new Promise((resolve) => server.close(resolve));
  return value;
}
function start(command, args, options = {}) {
  const child = spawn(command, args, {
    detached: true,
    stdio: ["ignore", "ignore", "pipe"],
    ...options,
  });
  child.on("error", (error) => {
    console.error(error);
    process.exitCode = 1;
  });
  child.stderr.on("data", () => {});
  children.push(child);
  return child;
}
async function ready(url) {
  for (let i = 0; i < 100; i++) {
    try {
      const r = await fetch(url);
      if (r.ok) return await r.json().catch(() => null);
    } catch {}
    await sleep(100);
  }
  throw Error(`Servidor não iniciou: ${url}`);
}
async function run() {
  const apiPort = await port(),
    vitePort = await port(),
    chromePort = await port();
  const base = `http://127.0.0.1:${apiPort}`;
  const bin = path.join(dir, "api");
  execFileSync("go", ["build", "-o", bin, "./cmd/api"], {
    cwd: path.join(root, "backend"),
  });
  const api = start(bin, [], {
    env: {
      ...process.env,
      GOOGLE_CLIENT_ID: process.argv.includes("--auth") ? "test-client" : "",
      GOOGLE_CLIENT_SECRET: process.argv.includes("--auth") ? "test-secret" : "",
      GOOGLE_REDIRECT_URI: process.argv.includes("--auth") ? `http://127.0.0.1:${vitePort}/api/auth/google/callback` : "",
      APP_URL: process.argv.includes("--auth") ? `http://127.0.0.1:${vitePort}` : "",
      PORT: String(apiPort),
      DATABASE_PATH: path.join(dir, "teste.db"),
    },
  });
  start(
    "node",
    [
      "node_modules/vite/bin/vite.js",
      "--host",
      "127.0.0.1",
      "--port",
      String(vitePort),
      "--strictPort",
    ],
    {
      cwd: path.join(root, "frontend"),
      env: { ...process.env, API_PROXY_TARGET: base },
    },
  );
  start(process.env.CHROME_BIN || "google-chrome", [
    "--headless=new",
    "--no-sandbox",
    "--disable-gpu",
    `--remote-debugging-port=${chromePort}`,
    `--user-data-dir=${path.join(dir, "chrome")}`,
    "about:blank",
  ]);
  await ready(`${base}/health`);
  await ready(`http://127.0.0.1:${vitePort}`);
  const targets = await ready(`http://127.0.0.1:${chromePort}/json`);
  ws = new WebSocket(
    targets.find((t) => t.type === "page").webSocketDebuggerUrl,
  );
  await new Promise((resolve, reject) => {
    ws.onopen = resolve;
    ws.onerror = reject;
  });
  let id = 0;
  let acceptDialogs = true;
  const dialogAnswers = [];
  const dialogs = [];
  const pending = new Map();
  function cdp(method, params = {}) {
    return new Promise((resolve, reject) => {
      const n = ++id;
      pending.set(n, { resolve, reject });
      ws.send(JSON.stringify({ id: n, method, params }));
    });
  }
  ws.onmessage = (event) => {
    const value = JSON.parse(event.data);
    if (value.method === "Page.javascriptDialogOpening") {
      dialogs.push(value.params.message);
      void cdp("Page.handleJavaScriptDialog", { accept: dialogAnswers.length ? dialogAnswers.shift() : acceptDialogs });
    }
    if (value.method === "Fetch.requestPaused") void cdp("Fetch.fulfillRequest", {
      requestId: value.params.requestId, responseCode: 200,
      responseHeaders: [{name:'Content-Type',value:'image/png'}],
      body:'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+a5ZkAAAAASUVORK5CYII=',
    });
    if (value.id) {
      const p = pending.get(value.id);
      pending.delete(value.id);
      value.error ? p.reject(value.error) : p.resolve(value.result);
    }
  };
  async function evaluate(expression) {
    const r = await cdp("Runtime.evaluate", {
      expression,
      awaitPromise: true,
      returnByValue: true,
    });
    if (r.exceptionDetails) throw Error(JSON.stringify(r.exceptionDetails));
    return r.result.value;
  }
  async function waitFor(expression) {
    for (let i = 0; i < 100; i++) {
      if (await evaluate(expression)) return;
      await sleep(100);
    }
    throw Error(
      `Timeout: ${expression}\n${await evaluate("document.body.innerText")}`,
    );
  }
  async function set(selector, value) {
    await waitFor(`!!document.querySelector(${JSON.stringify(selector)}) && !document.querySelector(${JSON.stringify(selector)}).matches(':disabled')`);
    await evaluate(
      `(()=>{const e=document.querySelector(${JSON.stringify(selector)});e.value=${JSON.stringify(value)};e.dispatchEvent(new Event('input',{bubbles:true}));e.dispatchEvent(new Event('change',{bubbles:true}));})()`,
    );
  }
  async function click(label) {
    await waitFor(`(()=>{const e=[...document.querySelectorAll('button')].find(e=>e.textContent.trim()===${JSON.stringify(label)});return !!e && !e.matches(':disabled') && e.checkVisibility()})()`);
    await evaluate(
      `(()=>{const e=[...document.querySelectorAll('button')].find(e=>e.textContent.trim()===${JSON.stringify(label)});if(!e||e.matches(':disabled')||!e.checkVisibility())throw Error('Botão indisponível: '+${JSON.stringify(label)});e.click()})()`,
    );
  }
  async function total(value) {
    await waitFor("!document.querySelector('#aba-historico').disabled");
    await click("Histórico");
    await waitFor(
      `document.body.innerText.includes(${JSON.stringify("Carga horária: " + value)}) && !document.body.innerText.includes('Carregando histórico…')`,
    );
  }
  async function request(method, route, body, status = 200) {
    const response = await fetch(base + route, {
      method,
      headers: { "Content-Type": "application/json" },
      ...(body ? { body: JSON.stringify(body) } : {}),
    });
    assert.equal(response.status, status);
    return response.json();
  }
  await cdp("Page.enable");
  if (process.argv.includes('--profile')) {
    await cdp('Fetch.enable',{patterns:[{urlPattern:'https://example.test/foto.png'}]});
    await cdp('Page.addScriptToEvaluateOnNewDocument',{source:`(() => {
      const original=window.fetch.bind(window);
      window.fetch=(url,options)=>String(url).endsWith('/auth/session')
        ? Promise.resolve(new Response(JSON.stringify({configurado:true,sheets_autorizado:true,usuario:{sub:'teste',nome:'Pessoa Teste',email:'oculto@example.test',foto:'https://example.test/foto.png'}}),{headers:{'Content-Type':'application/json'}}))
        : original(url,options);
    })()`});
  }
  if (process.argv.includes('--ux') || process.argv.includes('--sync')) await cdp('Page.addScriptToEvaluateOnNewDocument',{source:`(() => {
    const original=window.fetch.bind(window); window.__leituras=0;
    window.fetch=async (url,options)=>{
      if(String(url).endsWith('/integracoes/google/acesso')){
        window.__leituras++;
        if(window.__falharLeitura) return new Response(JSON.stringify({error:'Leitura recusada para teste.'}),{status:403,headers:{'Content-Type':'application/json'}});
        const config=await original('/api/integracoes/google/configuracao').then(r=>r.json());
        return new Response(JSON.stringify({titulo:'Planilha de teste',planilha_id:config.destino.planilha_id,versao:config.versao,leitura_confirmada:true,escrita_verificada:false}),{headers:{'Content-Type':'application/json'}});
      }
      return original(url,options);
    };
  })()`});
  if (process.argv.includes('--conteudos')) {
    await cdp('Page.addScriptToEvaluateOnNewDocument', {source:`(() => {
      const original = window.fetch.bind(window);
      window.__modoConteudos = 'aguardar';
      window.__criacaoConteudos = 0;
      window.fetch = async (url, options = {}) => {
        if (String(url).endsWith('/conteudos')) {
          if (options.method === 'POST') {
            window.__criacaoConteudos++;
            const response = await original(url, options);
            if (window.__modoConteudos === 'perder') throw new TypeError('Resposta perdida de teste');
            return response;
          }
          if (window.__modoConteudos === 'aguardar')
            await new Promise(resolve => { window.__liberarConteudos = () => {window.__modoConteudos = ''; resolve();}; });
          if (window.__modoConteudos === 'falhar') return new Response('{}', {status:503});
          if (window.__modoConteudos === 'invalido') return new Response('[{"id":1,"nome":""}]');
        }
        return original(url, options);
      };
    })()`});
  } else if (!process.argv.includes('--auth') && !process.argv.includes('--profile')) {
    // O formulário agora seleciona conteúdos persistidos, em vez de texto livre.
    for (const nome of ['Rascunho preservado entre abas', 'Conteúdo inicial', 'Minha edição preservada', 'Preservar durante falha', 'Sincronizar após cadastro', 'Salvar sem sincronizar'])
      await request('POST', '/conteudos', {nome}, 201);
  }
  await cdp("Page.navigate", { url: `http://127.0.0.1:${vitePort}` });
  if (process.argv.includes('--profile')) {
    await waitFor(`document.querySelector('.avatar')?.naturalWidth>0`);
    assert(await evaluate(`document.querySelector('.perfil').textContent.includes('Pessoa Teste')`));
    assert.equal(await evaluate(`document.body.textContent.includes('oculto@example.test')`),false);
    assert.equal(await evaluate(`getComputedStyle(document.querySelector('.avatar')).borderRadius`),'50%');
    assert.equal(await evaluate(`document.querySelector('.trocar-conta').textContent.trim()`),'Trocar de conta');
    assert.equal(await evaluate(`new URL(document.querySelector('.perfil form').action).pathname`),'/api/auth/google/login');
    await cdp('Emulation.setDeviceMetricsOverride',{width:375,height:812,deviceScaleFactor:1,mobile:true});
    assert(await evaluate('document.documentElement.scrollWidth<=window.innerWidth'));
    console.log('PASS: perfil com foto circular e nome, e-mail oculto, troca direta de conta e tela móvel. Sessão simulada.');
    return;
  }
  if (process.argv.includes("--auth")) {
    await waitFor(`document.body.innerText.includes('Entrar com Google')`);
    assert.equal(await evaluate("document.querySelector('#form-title')"), null);
    assert.equal(await evaluate("fetch('/api/aulas').then(r=>r.status)"), 401);
    assert.equal(await evaluate("document.querySelector('form').method"), "post");
    assert.equal(await evaluate("new URL(document.querySelector('form').action).pathname"), "/api/auth/google/login");
    await cdp("Page.navigate", { url: `http://127.0.0.1:${vitePort}/api/auth/google/callback?state=invalid&code=invalid` });
    await waitFor(`document.body.innerText.includes('Não foi possível concluir o login')`);
    assert.equal(await evaluate("location.search"), "");
    await cdp("Emulation.setDeviceMetricsOverride", { width:375, height:812, deviceScaleFactor:1, mobile:true });
    assert(await evaluate("document.documentElement.scrollWidth <= window.innerWidth"));
    console.log("PASS: tela de login, bloqueio da API, callback inválido, limpeza da URL e viewport móvel. Nenhuma chamada ao Google.");
    return;
  }
  await waitFor(`document.querySelector('#aba-cadastro')?.getAttribute('aria-selected')==='true'`);
  assert.equal(await evaluate("document.querySelectorAll('[role=tabpanel]:not([hidden])').length"), 1);
  if (process.argv.includes('--conteudos')) {
    acceptDialogs = false;
    await waitFor("typeof window.__liberarConteudos === 'function'");
    assert(await evaluate("document.querySelector('#conteudo').disabled && document.querySelector('button[type=submit]').disabled"));
    await evaluate('window.__liberarConteudos()');
    await waitFor("document.body.innerText.includes('Nenhum conteúdo cadastrado')");
    assert.equal(await evaluate("document.querySelector('#conteudo').tagName"), 'SELECT');
    await request('POST', '/turmas', {nome:'Turma de teste'}, 201);
    await click('Atualizar turmas');
    await set('#turma', '1');
    await set('#data', '2026-10-20');
    await set('#inicio', '08:00');
    await set('#fim', '09:00');
    await click('Salvar aula');
    await waitFor("document.querySelector('#conteudo-error')?.textContent.includes('Selecione')");
    assert.equal(await evaluate('document.activeElement.id'), 'conteudo');
    assert.equal((await request('GET', '/aulas')).total_registros, 0);

    const nome = 'Leitura\n  e revisão';
    await evaluate("document.querySelector('#cadastro-conteudo').open=true");
    await set('#novo-conteudo', nome);
    await evaluate("document.querySelector('#cadastro-conteudo button').click(); document.querySelector('#cadastro-conteudo button').click()");
    await waitFor(`document.querySelector('#conteudo').value === ${JSON.stringify(nome)} && !document.querySelector('#data').matches(':disabled')`);
    assert.equal(await evaluate('window.__criacaoConteudos'), 1);
    await waitFor("document.activeElement.id === 'conteudo'");
    assert.equal(await evaluate("document.querySelector('#data').value"), '2026-10-20');
    assert.equal(await evaluate("document.querySelector('#novo-conteudo').value"), '');
    assert.equal(await evaluate("document.querySelector('#conteudo-error')"), null);
    await click('Histórico'); await click('Registrar aula');
    assert.equal(await evaluate("document.querySelector('#conteudo').value"), nome);

    await set('#novo-conteudo', 'LEITURA E REVISÃO');
    await evaluate("document.querySelector('#cadastro-conteudo button').click()");
    await waitFor("document.querySelector('#cadastro-conteudo-error')?.textContent.includes('Já existe')");
    assert.equal(await evaluate("document.querySelector('#conteudo').value"), nome);
    assert(await evaluate("document.querySelector('#cadastro-conteudo button').disabled"));
    await click('Atualizar conteúdos');
    await waitFor("!document.querySelector('#cadastro-conteudo button').disabled");
    assert.equal((await request('GET', '/conteudos')).length, 1);

    await evaluate("window.__modoConteudos = 'perder'");
    await set('#novo-conteudo', 'Frações');
    await evaluate("document.querySelector('#cadastro-conteudo button').click()");
    await waitFor("document.querySelector('#cadastro-conteudo-error')?.textContent.includes('Não foi possível confirmar')");
    assert.equal((await request('GET', '/conteudos')).length, 2);
    assert.equal(await evaluate("document.querySelector('#novo-conteudo').value"), 'Frações');
    assert(await evaluate("document.querySelector('#cadastro-conteudo button').disabled"));
    await evaluate("window.__modoConteudos = ''");
    await click('Atualizar conteúdos');
    await waitFor("document.querySelectorAll('#conteudo option').length === 3 && !document.querySelector('#conteudo').disabled");
    assert.equal(await evaluate("document.querySelector('#conteudo').value"), nome);

    for (const modo of ['falhar', 'invalido']) {
      await evaluate(`window.__modoConteudos = ${JSON.stringify(modo)}`);
      await click('Atualizar conteúdos');
      await waitFor("!!document.querySelector('#conteudos-error')");
      assert(await evaluate("document.querySelector('button[type=submit]').disabled && document.querySelector('#conteudo').disabled"));
      assert.equal(await evaluate("document.querySelector('#conteudo').value"), nome);
      assert.equal(await evaluate("document.querySelector('#data').value"), '2026-10-20');
    }
    await evaluate("window.__modoConteudos = ''");
    await click('Atualizar conteúdos');
    await waitFor("!document.querySelector('#conteudo').disabled");
    await click('Salvar aula');
    await waitFor("document.querySelector('#conteudo').value === '' && !document.querySelector('#aba-historico').disabled");
    assert.equal((await request('GET', '/aulas/1')).conteudo, nome);
    await set('#turma', '1'); await set('#data', '2026-10-20');
    await set('#inicio', '09:00'); await set('#fim', '10:00');
    await set('#conteudo', nome); await click('Salvar aula');
    await waitFor("document.querySelector('#conteudo').value === '' && !document.querySelector('#aba-historico').disabled");
    assert.equal((await request('GET', '/aulas/2')).conteudo, nome);
    assert.equal((await request('GET', '/conteudos')).length, 2);

    // Texto antigo ausente do catálogo deve sobreviver à edição sem cadastro automático.
    const antiga = await request('POST', '/aulas', {data:'2026-10-21',inicio:'08:00',fim:'09:00',turma_id:1,conteudo:'Texto antigo\n  preservado',observacoes:'',intervalo_minutos:0}, 201);
    await click('Histórico');
    await click('Filtrar');
    await waitFor("document.querySelector('.aulas')?.textContent.includes('Texto antigo')");
    await click('Editar');
    await waitFor("document.querySelector('#form-title').textContent === 'Editar aula'");
    assert.equal(await evaluate("document.querySelector('#conteudo').value"), antiga.conteudo);
    assert(await evaluate("document.querySelector('#conteudo').selectedOptions[0].textContent.includes('conteúdo original')"));
    await set('#observacoes', 'Edição sem alterar conteúdo');
    await click('Salvar alterações');
    await waitFor("document.querySelector('#form-title').textContent === 'Dados da aula'");
    assert.equal((await request('GET', '/aulas/'+antiga.id)).conteudo, antiga.conteudo);
    assert.equal((await request('GET', '/conteudos')).length, 2);

    const pdf = await fetch(base + '/aulas/relatorio.pdf');
    assert.equal(pdf.status, 200);
    const pdfPath = path.join(dir, 'conteudos.pdf');
    fs.writeFileSync(pdfPath, Buffer.from(await pdf.arrayBuffer()));
    const textoPdf = execFileSync('pdftotext', [pdfPath, '-'], {encoding:'utf8'}).replace(/\s+/g, ' ');
    assert(textoPdf.includes('Leitura e revisão') && textoPdf.includes('Texto antigo preservado'));

    await cdp('Page.reload');
    await waitFor("typeof window.__liberarConteudos === 'function'");
    await evaluate('window.__liberarConteudos()');
    await waitFor("document.querySelectorAll('#conteudo option').length === 3 && !document.querySelector('#conteudo').disabled");
    await cdp('Emulation.setDeviceMetricsOverride', {width:375,height:812,deviceScaleFactor:1,mobile:true});
    await cdp('Emulation.setTouchEmulationEnabled', {enabled:true,maxTouchPoints:1});
    assert(await evaluate('document.documentElement.scrollWidth <= window.innerWidth'));
    await set('#conteudo', 'Frações');
    await click('Histórico'); await click('Registrar aula');
    assert.equal(await evaluate("document.querySelector('#conteudo').value"), 'Frações');
    console.log('PASS: catálogo vazio/carregando, conteúdo obrigatório, criação multilinha com seleção e foco, trava de clique duplo, duplicata, resposta perdida com recuperação, falha/JSON inválido, reutilização, edição antiga, persistência após recarga e viewport de toque.');
    return;
  }
  await set("#conteudo", "Rascunho preservado entre abas");
  await click("Histórico");
  await waitFor(`document.body.innerText.includes('Nenhuma aula encontrada')`);
  assert.equal(await evaluate("document.querySelector('#situacao').value"),'todas');
  await set('#situacao','ativas'); await click('Filtrar');
  await evaluate("document.querySelector('#aba-historico').focus()");
  await cdp("Input.dispatchKeyEvent", {type:"keyDown", key:"ArrowRight", code:"ArrowRight", windowsVirtualKeyCode:39});
  assert.equal(await evaluate("document.activeElement.id"), "aba-google");
  assert.equal(await evaluate("document.querySelector('#painel-google').hidden"), false);
  await waitFor("!document.querySelector('#google-url').matches(':disabled')");
  await set("#google-url", "https://docs.google.com/spreadsheets/d/rascunho/edit");
  await click("Registrar aula");
  assert.equal(await evaluate("document.querySelector('#conteudo').value"), "Rascunho preservado entre abas");
  await click("Integração Google");
  assert.equal(await evaluate("document.querySelector('#google-url').value"), "https://docs.google.com/spreadsheets/d/rascunho/edit");
  await evaluate("document.querySelector('#aba-google').focus()");
  await cdp("Input.dispatchKeyEvent", {type:"keyDown", key:"Home", code:"Home", windowsVirtualKeyCode:36});
  assert.equal(await evaluate("document.activeElement.id"), "aba-cadastro");
  await evaluate(`document.querySelector('details').open=true`);
  await set("#novo-nome", "Augusto");
  await click("Criar e selecionar");
  await waitFor(`document.querySelector('#turma').value==='1'`);
  await set("#data", "2026-10-18");
  await set("#inicio", "08:00");
  await set("#fim", "09:00");
  await set("#conteudo", "Conteúdo inicial");
  await click("Salvar aula");
  await total("1h00");
  await click("Editar");
  await waitFor(
    `document.querySelector('#form-title').textContent==='Editar aula' && !document.querySelector('#painel-cadastro').hidden`,
  );
  await set("#fim", "10:00");
  await click("Salvar alterações");
  await total("2h00");
  let aula = await request("GET", "/aulas/1");
  assert.equal(aula.versao, 2);
  assert.equal(aula.duracao_minutos, 120);
  await click("Editar");
  await set("#fim", "07:00");
  await click("Salvar alterações");
  await waitFor(
    `document.body.innerText.includes('O término deve ser posterior')`,
  );
  assert.equal((await request("GET", "/aulas/1")).versao, 2);
  // Simular outra aba que salvou a versão aberta no formulário.
  const corpo = {
    data: aula.data,
    inicio: aula.inicio,
    fim: aula.fim,
    turma_id: aula.turma_id,
    conteudo: "Alterado em outra aba",
    observacoes: "",
    intervalo_minutos: 0,
    versao: aula.versao,
  };
  await request("PUT", "/aulas/1", corpo);
  await set("#fim", "11:00");
  await set("#conteudo", "Minha edição preservada");
  await click("Salvar alterações");
  await waitFor(
    `document.body.innerText.includes('Esta aula mudou desde que foi aberta')`,
  );
  assert.equal(
    await evaluate(`document.querySelector('#conteudo').value`),
    "Minha edição preservada",
  );
  assert.equal(
    (await request("GET", "/aulas/1")).conteudo,
    "Alterado em outra aba",
  );
  await click("Fechar edição");
  await click("Histórico");
  await waitFor(
    `document.querySelector('.aulas')?.textContent.includes('Alterado em outra aba')`,
  );
  await click("Cancelar aula");
  await total("0h00");
  await set("#situacao", "canceladas");
  await click("Registrar aula");
  await click("Histórico");
  assert.equal(await evaluate("document.querySelector('#situacao').value"), "canceladas");
  await click("Filtrar");
  await waitFor(
    `document.querySelector('.aulas')?.textContent.includes('Cancelada')`,
  );
  assert.equal(
    await evaluate(
      `[...document.querySelectorAll('.aulas button')].some(e=>e.textContent==='Editar')`,
    ),
    false,
  );
  await click("Reativar aula");
  await waitFor(`document.body.innerText.includes('Nenhuma aula encontrada')`);
  await click("Limpar");
  await set("#situacao","ativas"); await click("Filtrar");
  await total("2h00");
  // Cancelar o único registro da última página deve voltar para a página anterior.
  for (let i = 0; i < 20; i++) {
    const { versao, ...novo } = corpo;
    await request("POST", "/aulas", novo, 201);
  }
  await click("Filtrar");
  await waitFor(`document.body.innerText.includes('21 aula(s)')`);
  await click("Próxima");
  await waitFor(`document.body.innerText.includes('Página 2 de 2')`);
  await click("Cancelar aula");
  await waitFor(
    `document.body.innerText.includes('Página 1 de 1') && document.querySelectorAll('#painel-historico .aulas li').length===20`,
  );
  await cdp("Emulation.setDeviceMetricsOverride", {
    width: 375,
    height: 900,
    deviceScaleFactor: 1,
    mobile: true,
  });
  assert(
    await evaluate("document.documentElement.scrollWidth <= window.innerWidth"),
  );
  await evaluate(`document.querySelector('#historico-title').scrollIntoView()`);
  const shot = await cdp("Page.captureScreenshot", { format: "png" });
  fs.writeFileSync(
    path.join(dir, "historico.png"),
    Buffer.from(shot.data, "base64"),
  );
  await cdp("Input.dispatchKeyEvent", {
    type: "keyDown",
    key: "Tab",
    code: "Tab",
    windowsVirtualKeyCode: 9,
  });
  await cdp("Input.dispatchKeyEvent", {
    type: "keyUp",
    key: "Tab",
    code: "Tab",
    windowsVirtualKeyCode: 9,
  });
  assert(await evaluate("document.activeElement!==document.body"));
  await click("Editar");
  await set("#conteudo", "Preservar durante falha");
  api.kill("SIGTERM");
  await sleep(500);
  await click("Salvar alterações");
  await waitFor(
    `document.body.innerText.includes('Os campos foram preservados')`,
  );
  assert.equal(
    await evaluate(`document.querySelector('#conteudo').value`),
    "Preservar durante falha",
  );
  start(bin, [], {
    env: {
      ...process.env,
      GOOGLE_CLIENT_ID: process.argv.includes("--auth") ? "test-client" : "",
      GOOGLE_CLIENT_SECRET: process.argv.includes("--auth") ? "test-secret" : "",
      GOOGLE_REDIRECT_URI: process.argv.includes("--auth") ? `http://127.0.0.1:${vitePort}/api/auth/google/callback` : "",
      APP_URL: process.argv.includes("--auth") ? `http://127.0.0.1:${vitePort}` : "",
      PORT: String(apiPort),
      DATABASE_PATH: path.join(dir, "teste.db"),
    },
  });
  await ready(`${base}/health`);
  await click("Salvar alterações");
  await waitFor(
    `document.querySelector('.aulas')?.textContent.includes('Preservar durante falha')`,
  );
  if (process.argv.includes("--google") || process.argv.includes("--sync") || process.argv.includes("--ux")) {
    await click("Integração Google");
    const link =
      process.env.GOOGLE_TEST_SHEET_URL ||
      "https://docs.google.com/spreadsheets/d/teste-planilha-A/edit?usp=sharing";
    const canonica = new URL(link);
    canonica.search = "";
    canonica.hash = "";
    await waitFor(
      `document.body.innerText.includes('Nenhuma planilha configurada.')`,
    );
    await set("#google-url", link);
    await click("Salvar destino");
    await waitFor(
      `document.body.innerText.includes('Destino salvo') && !document.querySelector('#google-url').matches(':disabled')`,
    );
    const config = await request("GET", "/integracoes/google/configuracao");
    assert.equal(config.destino.url, canonica.toString());
    assert.equal(config.versao, 1);
    // Aguarda o novo documento para não confundir o formulário antigo com
    // a configuração carregada após o reload.
    await evaluate("window.__googleAntesReload = true");
    await cdp("Page.reload");
    await waitFor(`!window.__googleAntesReload && !!document.querySelector('#aba-google')`);
    await click("Integração Google");
    await waitFor(`document.querySelector('#google-url')?.value===${JSON.stringify(canonica.toString())} && !document.querySelector('#google-url').matches(':disabled')`);
    await request("PUT", "/integracoes/google/configuracao", {
      url: "https://docs.google.com/spreadsheets/d/teste-planilha-B/edit",
      versao: 1,
    });
    await set("#google-url", link);
    await click("Salvar destino");
    await waitFor(
      `document.body.innerText.includes('A configuração mudou em outra aba')`,
    );
    assert.equal(
      await evaluate(`document.querySelector('#google-url').value`),
      link,
    );
    await click("Registrar aula");
    await click("Integração Google");
    await waitFor(`!document.querySelector('#google-url').matches(':disabled')`);
    await waitFor(
      `document.querySelector('.google-destino a')?.href.includes('teste-planilha-B')`,
    );
    await set("#google-url", "https://example.com/planilha");
    await click("Salvar destino");
    await waitFor(`document.body.innerText.includes('Informe um link HTTPS')`);
    await set("#google-url", link);
    await click("Salvar destino");
    await waitFor(
      `document.body.innerText.includes('Destino salvo') && !document.querySelector('#google-url').matches(':disabled')`,
    );
    for (const data of ["2026-11-01", "2027-10-01"]) {
      const { versao, ...novo } = corpo;
      await request("POST", "/aulas", { ...novo, data }, 201);
    }
    await click("Registrar aula");
    await click("Integração Google");
    await waitFor(`!document.querySelector('#google-url').matches(':disabled')`);
    await waitFor(
      `document.body.innerText.includes('Registros aulas - 10/2027') && document.body.innerText.includes('Registros aulas - 11/2026') && document.body.innerText.includes('Registros aulas - 10/2026')`,
    );
    assert(
      await evaluate(
        "document.documentElement.scrollWidth <= window.innerWidth",
      ),
    );
    await evaluate(`document.querySelector('#google-title').scrollIntoView()`);
    const googleShot = await cdp("Page.captureScreenshot", { format: "png" });
    fs.writeFileSync(
      path.join(dir, "google-config.png"),
      Buffer.from(googleShot.data, "base64"),
    );
    const plano = await request("GET", "/integracoes/google/planejamento");
    assert.equal(plano.verificacao_remota, "nao_realizada");
    assert.equal(plano.configuracao.destino.id, config.destino.id);
    console.log(
      "PASS: configuração Google local, link normalizado, persistência, conflito, troca de destino, validação e prévia mensal. Nenhuma chamada ao Google.",
    );
  }
  if (process.argv.includes("--sync") || process.argv.includes("--ux")) {
    // Fixtures só no navegador: validam controles e contratos sem credenciais reais.
    // O backend/adaptador é exercitado separadamente nos testes Go.
    const destino = await request("GET", "/integracoes/google/configuracao");
    await evaluate(`(() => {
      const original = window.fetch.bind(window);
      const config = ${JSON.stringify(destino)};
      window.__syncFixture = { prepared: false, synced: [], versoes: {}, attempts: 0, requests: [] };
      window.fetch = async (input, options = {}) => {
        const url = String(input);
        const f = window.__syncFixture;
        const json = (body, status = 200) => new Response(JSON.stringify(body), {status, headers:{'Content-Type':'application/json'}});
        if (url.endsWith('/integracoes/google/inspecionar') || url.endsWith('/integracoes/google/preparar')) {
          const body = JSON.parse(options.body); f.requests.push({url, body});
          if (url.endsWith('/preparar')) {
            if (body.confirmacao !== 'fixture-token') throw Error('Confirmação ausente');
            f.prepared = true;
          }
          return json({periodo: body.periodo, aba:'Registros aulas - ' + body.periodo.slice(5) + '/' + body.periodo.slice(0,4), config_versao:config.versao, planilha_id:config.destino.planilha_id, estado:f.prepared?'pronta':'ausente', mensagem:f.prepared?'Cabeçalhos compatíveis.':'Confirme a criação da aba mensal.', ...(f.prepared?{}:{confirmacao:'fixture-token'})});
        }
        if (url.endsWith('/sincronizar')) {
          f.requests.push({url, body:JSON.parse(options.body)}); f.attempts++;
          if (f.attempts === 1 || f.attempts === f.failAt) return json({error:'Falha simulada no Google; tente novamente.'},409);
          f.synced.push(Number(url.split('/').at(-2))); f.versoes[Number(url.split('/').at(-2))]=JSON.parse(options.body).versao; return json({estado:'sincronizado'});
        }
        if (options.method === 'DELETE' && f.excluirGoogleFixture) {
          const body=JSON.parse(options.body); f.exclusoes.push(body);
          if (body.remover_google && f.exclusoes.length===1) return json({error:'Sem permissão para excluir na planilha de teste.'},403);
          return original(input,{...options,body:JSON.stringify({...body,remover_google:false})});
        }
        const res = await original(input,options);
        if (url.includes('/aulas?')) {
          const body=await res.json();
          for (const aula of body.aulas) aula.google={estado:f.versoes[aula.id]===aula.versao?'sincronizado':f.attempts?'erro':f.prepared?'pendente':'aguardando_preparacao',destino_id:config.destino.id,config_versao:config.versao,periodo:aula.data.slice(0,7),tentativas:f.attempts};
          return json(body,res.status);
        }
        return res;
      };
    })()`);
    await click("Inspecionar aba mensal");
    await waitFor(`document.body.innerText.includes('Confirme a criação da aba mensal.')`);
    await click("Agora não");
    assert.equal(await evaluate("window.__syncFixture.prepared"),false);
    await click("Inspecionar aba mensal");
    await waitFor(`document.body.innerText.includes('Confirmar preparação de Registros aulas - 10/2027')`);
    await click("Confirmar preparação de Registros aulas - 10/2027");
    await waitFor(`document.body.innerText.includes('Cabeçalhos compatíveis.')`);
    await click("Histórico");
    await set("#situacao","ativas"); await click("Filtrar");
    await waitFor(`document.body.innerText.includes('Envio pendente')`);
    await evaluate("window.__syncFixture.prepared=false");
    dialogAnswers.push(true,false);
    await click("Sincronizar com Google");
    await waitFor(`document.body.innerText.includes('Aula preservada. Envio pendente') && !document.querySelector('#aba-historico').disabled`);
    assert.equal(await evaluate('window.__syncFixture.attempts'),0);
    await click("Sincronizar com Google");
    await waitFor(`document.body.innerText.includes('Falha simulada no Google') && document.body.innerText.includes('Tentar envio novamente')`);
    await click("Tentar envio novamente");
    await waitFor(`document.body.innerText.includes('Sincronizado')`);
    const requests = await evaluate("window.__syncFixture.requests");
    assert(requests.length >= 5);
    assert(requests.every(r=>r.body.config_versao===destino.versao));
    const envios=requests.filter(r=>r.url.endsWith('/sincronizar'));
    assert.equal(envios[0].body.versao,envios[1].body.versao);
    assert(await evaluate("document.documentElement.scrollWidth <= window.innerWidth"));
    console.log("PASS: interface de inspeção, recusa, confirmação vinculada ao destino, atualização do histórico, erro e nova tentativa; respostas Sheets simuladas no navegador.");
    if (process.argv.includes("--ux")) {
      assert(dialogs.some(message=>message.includes('Deseja sincronizar com o Google agora?')));
      assert(dialogs.some(message=>message.includes('não existe. Deseja criá-la')));
      const antesReenvio=await evaluate('window.__syncFixture.requests.length');
      await click('Sincronizar com Google');
      await waitFor(`document.body.innerText.includes('Esta aula já está na planilha.')`);
      assert.equal(await evaluate('window.__syncFixture.requests.length'),antesReenvio);
      assert.equal(await evaluate("[...document.querySelectorAll('button')].some(b=>b.textContent.trim()==='Verificar leitura com minha conta')"),false);
      assert(await evaluate('window.__leituras>0'));

      assert.equal(await evaluate("[...document.querySelectorAll('button')].some(b=>b.textContent.includes('Recarregar configuração e prévia'))"),false);
      assert.equal(await evaluate("document.querySelector('#painel-google').textContent.includes('Abas esperadas por mês')"),false);
      assert(await evaluate("document.querySelectorAll('.aula-card.sincronizado').length===1 && document.querySelectorAll('.aula-card.envio-pendente').length>0"));
      // Recusar um lote não executa envio algum.
      acceptDialogs = false;
      const before = await evaluate("window.__syncFixture.requests.length");
      await click("Enviar todas as pendentes");
      await waitFor(`document.body.innerText.includes('Nenhuma alteração realizada.') && !document.querySelector('#aba-historico').disabled`);
      assert.equal(await evaluate("window.__syncFixture.requests.length"),before);
      acceptDialogs = true;
      await evaluate("window.__syncFixture.failAt=window.__syncFixture.attempts+2");
      await click("Enviar todas as pendentes");
      await waitFor(`document.body.innerText.includes('Lote interrompido') && !document.querySelector('#aba-historico').disabled`);
      assert(dialogs.at(-1).includes('Todas as páginas'));
      await evaluate("window.__syncFixture.failAt=0");
      await click("Enviar todas as pendentes");
      await waitFor(`document.body.innerText.includes('sincronizada(s).') && !document.querySelector('#aba-historico').disabled`);
      const all = await request("GET", "/aulas?situacao=ativas");
      assert(all.total_registros > 20);
      const synced = await evaluate("window.__syncFixture.synced");
      assert.equal(new Set(synced).size,all.total_registros);
      assert.equal(synced.length,all.total_registros);
      // Filtro limita o lote, incluindo a contagem mostrada na confirmação.
      await set('#data-inicial','2027-01-01');
      await click('Filtrar');
      await waitFor(`document.body.innerText.includes('1 aula(s) no filtro')`);
      acceptDialogs=false;
      await click('Cancelar todas as aulas');
      await waitFor(`document.body.innerText.includes('Nenhuma alteração realizada.') && !document.querySelector('#aba-historico').disabled`);
      assert.equal((await request('GET','/aulas?situacao=ativas')).total_registros,all.total_registros);
      acceptDialogs=true;
      await click('Cancelar todas as aulas');
      await waitFor(`document.body.innerText.includes('Nenhuma aula encontrada') && !document.querySelector('#aba-historico').disabled`);
      assert(dialogs.at(-1).includes('Cancelar 1 aula(s)'));
      await click('Limpar');
      await set('#situacao','ativas'); await click('Filtrar');
      await waitFor(`document.body.innerText.includes('${all.total_registros-1} aula(s) no filtro')`);
      await click('Cancelar todas as aulas');
      await waitFor(`document.body.innerText.includes('Nenhuma aula encontrada') && !document.querySelector('#aba-historico').disabled`);
      assert.equal((await request('GET','/aulas?situacao=ativas')).total_registros,0);
      assert.equal((await request('GET','/aulas?situacao=todas')).total_registros,all.total_registros+1);
      assert(await evaluate("document.documentElement.scrollWidth <= window.innerWidth"));
      await click('Registrar aula');
      await set('#data','2026-10-20'); await set('#inicio','08:00'); await set('#fim','09:00'); await set('#turma','1'); await set('#conteudo','Sincronizar após cadastro');
      await click('Salvar aula');
      await waitFor(`document.body.innerText.includes('Aula salva e sincronizada com o Google.') && !document.querySelector('#aba-historico').disabled`);
      assert.equal((await evaluate('window.__syncFixture.synced')).length,all.total_registros+1);
      acceptDialogs=false;
      await set('#data','2026-10-20'); await set('#inicio','10:00'); await set('#fim','11:00'); await set('#turma','1'); await set('#conteudo','Salvar sem sincronizar');
      await click('Salvar aula');
      await waitFor(`document.body.innerText.includes('Aula salva.') && !document.querySelector('#aba-historico').disabled`);
      assert.equal((await evaluate('window.__syncFixture.synced')).length,all.total_registros+1);
      assert.equal((await request('GET','/aulas?situacao=ativas')).total_registros,2);
      acceptDialogs=true;
      await click('Histórico'); await set('#situacao','canceladas'); await click('Filtrar');
      await waitFor(`document.querySelectorAll('.aula-card.cancelada').length>0 && !document.body.innerText.includes('Carregando histórico…')`);
      assert.equal(await evaluate("document.querySelectorAll('.aula-card.cancelada.sincronizado').length"),0);
      assert(await evaluate("document.body.innerText.includes('Cancelamento pendente de sincronização')"));
      const antesExcluir=(await request('GET','/aulas?situacao=todas')).total_registros;
      await click('Excluir registro');
      await waitFor(`document.querySelector('dialog')?.open`);
      assert.equal(await evaluate("document.querySelector('dialog input[type=checkbox]').checked"),false);
      await click('Voltar');
      assert.equal((await request('GET','/aulas?situacao=todas')).total_registros,antesExcluir);
      await click('Excluir registro'); await waitFor(`document.querySelector('dialog')?.open`);
      await click('Excluir definitivamente');
      await waitFor(`!document.querySelector('dialog').open && document.body.innerText.includes('Os registros no Google foram preservados.')`);
      assert.equal((await request('GET','/aulas?situacao=todas')).total_registros,antesExcluir-1);
      await waitFor(`!document.body.innerText.includes('Carregando histórico…')`);
      await evaluate('window.__syncFixture.excluirGoogleFixture=true; window.__syncFixture.exclusoes=[]');
      await click('Excluir registro'); await waitFor(`document.querySelector('dialog').open`);
      await evaluate("document.querySelector('dialog input[type=checkbox]').click()");
      await click('Excluir definitivamente');
      await waitFor(`document.body.innerText.includes('Sem permissão para excluir') && !document.querySelector('#aba-historico').disabled`);
      assert(await evaluate("document.querySelector('dialog').open && document.querySelector('dialog input').checked"));
      assert.equal((await request('GET','/aulas?situacao=todas')).total_registros,antesExcluir-1);
      await click('Excluir definitivamente');
      await waitFor(`!document.querySelector('dialog').open && document.body.innerText.includes('Aula excluída definitivamente do aplicativo e da planilha configurada.')`);
      const exclusoes=await evaluate('window.__syncFixture.exclusoes');
      assert.equal(exclusoes.length,2);assert(exclusoes.every(e=>e.remover_google===true));
      assert.equal((await request('GET','/aulas?situacao=todas')).total_registros,antesExcluir-2);

      // Falha de leitura posterior ao salvamento conserva a configuração salva.
      await click('Integração Google'); await waitFor(`!document.querySelector('#google-url').matches(':disabled')`);
      await evaluate('window.__falharLeitura=true');
      await click('Salvar destino');
      await waitFor(`document.body.innerText.includes('Destino salvo, mas não foi possível confirmar o acesso') && !document.querySelector('#google-url').matches(':disabled')`);
      const configDepois=await request('GET','/integracoes/google/configuracao');
      assert.equal(configDepois.versao,destino.versao);
      await evaluate('window.__falharLeitura=false');
      await click('Salvar destino');
      await waitFor(`document.body.innerText.includes('Destino salvo e acesso de leitura confirmado.') && !document.querySelector('#google-url').matches(':disabled')`);
      console.log('PASS: aviso de reenvio sem chamadas, criação de aba pelo histórico aceita/recusada, canceladas vermelhas, exclusão definitiva local e verificação automática de leitura com falha recuperável.');
      console.log('PASS: convite após cadastro, estados por cor/texto, recusa dos lotes, falha parcial, envio e cancelamento em múltiplas páginas, filtros e preservação dos registros.');
    }

  }
  await evaluate("window.scrollTo(0,0)");
  const tabsShot = await cdp("Page.captureScreenshot", { format: "png" });
  fs.writeFileSync(path.join(dir, "abas.png"), Buffer.from(tabsShot.data, "base64"));
  console.log(
    "PASS: abas, teclado, preservação de rascunhos/filtros, cadastro, edição, recálculo, validação, conflito, cancelamento, reativação, paginação, celular, foco e recuperação.",
  );
  console.log(`Artefatos temporários: ${dir}`);
}
run()
  .catch((error) => {
    console.error(error);
    process.exitCode = 1;
  })
  .finally(async () => {
    ws?.close();
    for (const child of children.reverse()) {
      if (!child.pid) continue;
      try {
        process.kill(-child.pid, "SIGTERM");
      } catch {}
    }
    await sleep(500);
  });
