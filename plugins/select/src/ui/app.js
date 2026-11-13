'use strict';
const themeToggle = document.querySelector('#theme-toggle');
const systemTheme = window.matchMedia('(prefers-color-scheme: dark)');
let preferredTheme = null;
try { preferredTheme = localStorage.getItem('select-theme') ?? localStorage.getItem('db-plugin-theme'); } catch { /* System theme works without local storage. */ }
if (preferredTheme === 'light' || preferredTheme === 'dark') document.documentElement.dataset.theme = preferredTheme;
function updateThemeLabel() {
  const dark = document.documentElement.dataset.theme === 'dark' || (!document.documentElement.dataset.theme && systemTheme.matches);
  const label = `Switch to ${dark ? 'light' : 'dark'} theme`;
  themeToggle.setAttribute('aria-label', label);
  themeToggle.title = label;
}
themeToggle.addEventListener('click', () => {
  const dark = document.documentElement.dataset.theme === 'dark' || (!document.documentElement.dataset.theme && systemTheme.matches);
  const theme = dark ? 'light' : 'dark';
  document.documentElement.dataset.theme = theme;
  try { localStorage.setItem('select-theme', theme); } catch { /* Keep the theme for this window. */ }
  updateThemeLabel();
});
systemTheme.addEventListener('change', updateThemeLabel);
updateThemeLabel();
const newProjectButton = document.querySelector('#new-project');
const form = document.querySelector('#connection');
const fields = form.elements;
const projectStatus = document.querySelector('#project-status');
const status = document.querySelector('#status');
const saveState = document.querySelector('#save-state');
const saveButton = document.querySelector('#save');
const saveLabel = document.querySelector('#save-label');
const testButton = document.querySelector('#test-connection');
const actions = document.querySelector('.form-actions');
const environmentSelect = document.querySelector('#environment-select');
let projects = Object.create(null);
let selectedProject = '';
let selectedEnvironment = '';
let ready = false;
let uiBusy = false;
let baseline = null;
let savedTimer = null;
const has = (object, key) => Object.prototype.hasOwnProperty.call(object, key);
const validName = name => name.trim().length > 0 && !/[\u0000-\u001f\u007f]/.test(name);
const api = window.go.main.App;
const projectOptions = document.querySelector('#project-options');
const environmentOptions = document.querySelector('#environment-options');
const projectMenu = document.querySelector('#project-menu');
const environmentMenu = document.querySelector('#environment-menu');
const projectSummary = document.querySelector('#project-summary');
const manageDialog = document.querySelector('#manage-dialog');
const manageForm = document.querySelector('#manage-form');
const manageName = document.querySelector('#manage-name');
const manageError = document.querySelector('#manage-error');
const manageConfirm = document.querySelector('#manage-confirm');
const manageCancel = document.querySelector('#manage-cancel');
let management = null;
const projectColors = ['', 'blue', 'green', 'purple', 'orange', 'red', 'pink', 'teal', 'amber'];
for (const color of projectColors) {
  const button = document.createElement('button');
  const name = color ? color[0].toUpperCase() + color.slice(1) : 'Default';
  button.type = 'button'; button.className = 'color-swatch'; button.dataset.color = color;
  button.setAttribute('aria-label', `${name} project color`); button.title = name;
  button.setAttribute('aria-pressed', 'false');
  button.addEventListener('click', () => changeProjectColor(color));
  document.querySelector('#project-colors').append(button);
}
const environmentEmojis = { neutral: '⚪', blue: '🔵', orange: '🟠', red: '🔴', green: '🟢', yellow: '🟡', purple: '🟣' };
function defaultEnvironmentColor(name) {
  switch (name.trim().toLowerCase()) {
    case 'development': case 'dev': return 'blue';
    case 'staging': case 'stage': return 'orange';
    case 'production': case 'prod': return 'red';
    default: return 'neutral';
  }
}
for (const [color, emoji] of Object.entries(environmentEmojis)) {
  const button = document.createElement('button');
  const name = color[0].toUpperCase() + color.slice(1);
  button.type = 'button'; button.className = 'color-swatch'; button.dataset.color = color;
  button.setAttribute('aria-label', `${name} environment color`); button.title = `${emoji} ${name}`;
  button.setAttribute('aria-pressed', 'false');
  button.addEventListener('click', () => changeEnvironmentColor(color));
  document.querySelector('#environment-colors').append(button);
}
function updateEnvironmentAppearance() {
  const color = projects[selectedProject]?.environments[selectedEnvironment]?.color || 'neutral';
  for (const button of document.querySelectorAll('#environment-colors .color-swatch')) button.setAttribute('aria-pressed', String(button.dataset.color === color));
}
async function changeEnvironmentColor(color) {
  const project = selectedProject, name = selectedEnvironment;
  if (uiBusy || !project || !name || projects[project].environments[name].color === color) return;
  closeMenus(); setBusy(true);
  try {
    await api.SetEnvironmentColor(project, name, color);
    projects[project].environments[name].color = color;
    renderEnvironments(name); updateEnvironmentAppearance();
    setFeedback('Environment color updated.', 'success');
  } catch (error) { setFeedback(String(error?.message || error), 'error'); }
  finally { setBusy(false); }
}
function closeMenus() {
  projectMenu.hidden = true; environmentMenu.hidden = true;
  projectOptions.setAttribute('aria-expanded', 'false'); environmentOptions.setAttribute('aria-expanded', 'false');
}
function toggleMenu(menu, toggle) {
  const opening = menu.hidden;
  closeMenus();
  if (opening) {
    menu.hidden = false; toggle.setAttribute('aria-expanded', 'true');
    menu.querySelector('button').focus();
  }
}
projectOptions.addEventListener('click', () => toggleMenu(projectMenu, projectOptions));
environmentOptions.addEventListener('click', () => toggleMenu(environmentMenu, environmentOptions));
document.addEventListener('click', event => { if (!event.target.closest?.('.options-anchor')) closeMenus(); });
document.addEventListener('keydown', event => {
  if (event.key !== 'Escape') return;
  const toggle = !projectMenu.hidden ? projectOptions : !environmentMenu.hidden ? environmentOptions : null;
  closeMenus(); toggle?.focus();
});
function updateProjectAppearance() {
  const color = projects[selectedProject]?.color || '';
  projectSummary.dataset.color = color;
  for (const button of document.querySelectorAll('#project-colors .color-swatch')) button.setAttribute('aria-pressed', String(button.dataset.color === color));
}
function updateManagementControls() {
  const projectMethods = ['RenameProject', 'DeleteProject', 'SetProjectColor'].every(name => typeof api[name] === 'function');
  const environmentMethods = ['RenameEnvironment', 'DeleteEnvironment', 'SetEnvironmentColor'].every(name => typeof api[name] === 'function');
  projectOptions.disabled = uiBusy || !ready || !selectedProject || !projectMethods;
  environmentOptions.disabled = uiBusy || !ready || !selectedEnvironment || !environmentMethods;
  projectOptions.title = projectMethods ? 'Project options' : 'Close and reopen settings to enable project options.';
  environmentOptions.title = environmentMethods ? 'Environment options' : 'Close and reopen settings to enable environment options.';
  manageName.disabled = uiBusy || !management || management.type.startsWith('delete');
  manageConfirm.disabled = uiBusy || !management;
}
async function changeProjectColor(color) {
  const project = selectedProject;
  if (uiBusy || !project || projects[project].color === color) return;
  closeMenus(); setBusy(true);
  try {
    await api.SetProjectColor(project, color);
    projects[project].color = color;
    renderProjects(); updateProjectAppearance();
    setFeedback('Project color updated.', 'success');
  } catch (error) { setFeedback(String(error?.message || error), 'error'); }
  finally { setBusy(false); }
}
function openManagement(type) {
  closeMenus();
  management = { type, project: selectedProject, environment: selectedEnvironment };
  const creating = type === 'create-project';
  const project = type.endsWith('project');
  const removing = type.startsWith('delete');
  const name = creating ? '' : project ? selectedProject : selectedEnvironment;
  document.querySelector('#manage-title').textContent = creating ? 'New project' : `${removing ? 'Delete' : 'Rename'} ${project ? 'project' : 'environment'}`;
  document.querySelector('#manage-description').hidden = creating;
  document.querySelector('#manage-description').textContent = creating ? '' : removing
    ? project ? `Delete “${name}” and all its saved environments from Select? Databases will not be changed.` : `Delete “${name}” from Select? The database will not be changed.`
    : `Choose a new name for “${name}”.`;
  if (removing && hasChanges()) document.querySelector('#manage-description').textContent += ' Unsaved connection edits will also be discarded.';
  document.querySelector('#rename-field').hidden = removing;
  document.querySelector('#manage-name-label').textContent = creating ? 'Project name' : 'New name';
  manageName.value = name; manageError.textContent = '';
  manageConfirm.textContent = creating ? 'Create' : removing ? 'Delete' : 'Rename'; manageConfirm.className = removing ? 'danger' : 'primary';
  updateManagementControls(); manageDialog.showModal();
  if (removing) manageCancel.focus(); else { manageName.focus(); manageName.select(); }
}
newProjectButton.addEventListener('click', () => openManagement('create-project'));
for (const type of ['rename-project', 'delete-project', 'rename-environment', 'delete-environment']) {
  document.querySelector(`#${type}`).addEventListener('click', () => openManagement(type));
}
manageCancel.addEventListener('click', () => manageDialog.close());
manageDialog.addEventListener('cancel', event => { if (uiBusy) event.preventDefault(); });
manageDialog.addEventListener('close', () => { management = null; });
manageForm.addEventListener('submit', async event => {
  event.preventDefault();
  if (!management || uiBusy) return;
  const action = management;
  const newName = manageName.value.trim();
  if (!action.type.startsWith('delete') && !validName(newName)) { manageError.textContent = 'Enter a name.'; return; }
  let created = false;
  setBusy(true); manageError.textContent = '';
  try {
    if (action.type === 'create-project') {
      await api.CreateProject(newName);
      await loadProjects();
      selectProject(newName); created = true;
    } else if (action.type === 'rename-project') {
      await api.RenameProject(action.project, newName);
      if (newName !== action.project) { projects[newName] = projects[action.project]; delete projects[action.project]; }
      selectedProject = newName; document.querySelector('#project-title').textContent = newName;
      renderProjects(); updateProjectAppearance();
    } else if (action.type === 'delete-project') {
      await api.DeleteProject(action.project);
      delete projects[action.project]; selectProject(Object.keys(projects).sort()[0] || '');
    } else if (action.type === 'rename-environment') {
      await api.RenameEnvironment(action.project, action.environment, newName);
      const environments = projects[action.project].environments;
      if (newName !== action.environment) { environments[newName] = environments[action.environment]; delete environments[action.environment]; }
      selectedEnvironment = newName; fields.environment.value = newName;
      const saved = JSON.parse(baseline); saved[0] = newName; baseline = JSON.stringify(saved);
      renderEnvironments(newName); document.querySelector('#connection-title').textContent = `${newName} connection`;
      renderProjects(); updateDirtyState();
    } else {
      await api.DeleteEnvironment(action.project, action.environment);
      delete projects[action.project].environments[action.environment]; selectProject(action.project);
    }
    manageDialog.close();
    const message = created ? 'Project created.' : action.type.startsWith('delete') ? 'Deleted from Select.' : 'Renamed.';
    if (selectedProject) setFeedback(message, 'success'); else projectStatus.textContent = message;
  } catch (error) { manageError.textContent = String(error?.message || error) || 'Could not update settings.'; }
  finally { setBusy(false); if (created) fields.host.focus(); }
});
function formSnapshot() {
  return JSON.stringify(['environment', 'host', 'port', 'database', 'user', 'password'].map(name => fields[name].value).concat(fields.tls.checked));
}
function hasChanges() { return baseline !== null && formSnapshot() !== baseline; }
function setFeedback(message, kind = '') {
  status.textContent = message;
  status.dataset.kind = kind;
}
function updateSaveControls() {
  saveButton.disabled = uiBusy || !ready || !hasChanges();
  testButton.disabled = uiBusy || !ready || !selectedProject || typeof api.TestConnection !== 'function';
  testButton.title = typeof api.TestConnection === 'function' ? '' : 'Close and reopen settings to enable connection testing.';
  updateManagementControls();
}
function updateDirtyState() {
  window.clearTimeout(savedTimer);
  const changed = hasChanges();
  saveState.textContent = changed ? 'Unsaved changes' : '';
  updateSaveControls();
}
for (const event of ['input', 'change']) form.addEventListener(event, () => {
  setFeedback('');
  updateDirtyState();
});
async function watchStyles() {
  let previousCSS = null;
  async function check() {
    try {
      if (!document.hidden) {
        const url = `/style.css?preview=${Date.now()}`;
        const response = await fetch(url, { cache: 'no-store' });
        if (response.ok) {
          const css = await response.text();
          if (css !== previousCSS) {
            document.querySelector('#app-style').href = url;
            previousCSS = css;
          }
        }
      }
    } catch { /* Retry after a temporary file-save or reload interruption. */ }
    window.setTimeout(check, 1000);
  }
  await check();
}
if (typeof api.DevelopmentMode === 'function') {
  api.DevelopmentMode().then(development => {
    if (!development) return;
    watchStyles();
  }).catch(() => { /* Embedded production assets work without live preview. */ });
}
function setBusy(busy, operation = '') {
  uiBusy = busy;
  actions.setAttribute('aria-busy', String(busy));
  for (const element of document.querySelectorAll('button:not(#theme-toggle),input,select')) element.disabled = busy || !ready;
  saveLabel.textContent = busy && operation === 'save' ? 'Saving…' : 'Save';
  testButton.textContent = busy && operation === 'test' ? 'Testing…' : 'Test connection';
  updateSaveControls();
}
function renderProjects() {
  const list = document.querySelector('#projects');
  list.replaceChildren();
  for (const project of Object.keys(projects).sort()) {
    const button = document.createElement('button');
    button.type = 'button'; button.className = 'project'; button.setAttribute('aria-pressed', String(project === selectedProject));
    button.dataset.color = projects[project].color || '';
    const title = document.createElement('span'); title.textContent = project;
    const count = document.createElement('small');
    const total = Object.keys(projects[project].environments).length;
    count.textContent = `${total} environment${total === 1 ? '' : 's'}`;
    const icon = document.createElement('span'); icon.className = 'project-icon'; icon.setAttribute('aria-hidden', 'true');
    icon.innerHTML = '<svg viewBox="0 0 24 24"><path d="M3 7a2 2 0 0 1 2-2h5l2 2h7a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2Z"/></svg>';
    const copy = document.createElement('span'); copy.className = 'project-copy'; copy.append(title, count);
    button.append(icon, copy);
    button.addEventListener('click', () => selectProject(project)); list.append(button);
  }
}
function selectProject(project) {
  closeMenus();
  selectedProject = project; projectStatus.textContent = ''; renderProjects();
  document.querySelector('#empty').hidden = Boolean(project);
  document.querySelector('#editor').hidden = !project;
  if (!project) { baseline = null; updateDirtyState(); return; }
  document.querySelector('#project-title').textContent = project;
  updateProjectAppearance();
  const names = renderEnvironments('');
  selectEnvironment(names[0] || '');
}
function renderEnvironments(selected) {
  const project = selectedProject;
  const order = name => {
    switch (name.trim().toLowerCase()) {
      case 'development': case 'dev': return 0;
      case 'staging': case 'stage': return 1;
      case 'production': case 'prod': return 2;
      default: return 3;
    }
  };
  const names = Object.keys(projects[project].environments).sort((a, b) => order(a) - order(b) || a.localeCompare(b));
  environmentSelect.replaceChildren();
  for (const name of names) {
    const option = document.createElement('option'); option.value = name; option.textContent = `${environmentEmojis[projects[project].environments[name].color] || '⚪'} ${name}`; environmentSelect.append(option);
  }
  const add = document.createElement('option'); add.value = ''; add.textContent = '+ Add environment'; environmentSelect.append(add);
  environmentSelect.value = selected;
  return names;
}
function selectEnvironment(environment) {
  closeMenus();
  selectedEnvironment = environment; environmentSelect.value = environment; form.reset(); setFeedback('');
  const connection = environment ? projects[selectedProject].environments[environment].connection : undefined;
  updateEnvironmentAppearance();
  fields.environment.value = environment; fields.environment.readOnly = Boolean(environment);
  document.querySelector('#connection-title').textContent = environment ? `${environment} connection` : 'New environment';
  for (const field of ['host', 'port', 'database', 'user']) fields[field].value = connection?.[field] ?? (field === 'port' ? 3306 : '');
  fields.tls.checked = connection?.tls?.enabled ?? true;
  fields.password.value = '';
  fields.password.placeholder = connection ? 'Leave blank to keep saved password' : 'Database password';
  baseline = formSnapshot();
  updateDirtyState();
}
environmentSelect.addEventListener('change', () => selectEnvironment(environmentSelect.value));
function connectionInput() {
  if (!form.reportValidity()) return null;
  const project = selectedProject;
  const environment = fields.environment.value.trim();
  if (!project || !validName(environment)) { setFeedback('Select a project and enter a valid environment name.', 'error'); return null; }
  if (!selectedEnvironment && has(projects[project].environments, environment)) { setFeedback('This environment already exists. Select it above to edit it.', 'error'); return null; }
  const connection = { host: fields.host.value.trim(), port: Number(fields.port.value), database: fields.database.value.trim(), user: fields.user.value.trim(), tls: { enabled: fields.tls.checked } };
  const savedCA = selectedEnvironment ? projects[project].environments[selectedEnvironment].connection?.tls?.caFile : undefined;
  if (connection.tls.enabled && savedCA) connection.tls.caFile = savedCA;
  const body = { project, environment, connection };
  if (!projects[project].environments[selectedEnvironment]?.connection || fields.password.value !== '') body.password = fields.password.value;
  return body;
}
testButton.addEventListener('click', async () => {
  if (uiBusy) return;
  const body = connectionInput();
  if (!body) return;
  setBusy(true, 'test'); setFeedback('Testing connection…');
  try {
    await api.TestConnection(body);
    setFeedback('Connected', 'success');
  } catch (error) {
    setFeedback(`Connection failed: ${String(error?.message || error) || 'Check connection settings.'}`, 'error');
  } finally { setBusy(false); }
});
form.addEventListener('submit', async event => {
  event.preventDefault();
  if (uiBusy || !hasChanges()) return;
  const body = connectionInput();
  if (!body) return;
  const { project, environment, connection } = body;
  setBusy(true, 'save'); setFeedback('');
  try {
    await api.SaveEnvironment(body);
    projects[project].environments[environment] = { color: projects[project].environments[environment]?.color || defaultEnvironmentColor(environment), connection };
    selectProject(project); selectEnvironment(environment);
    saveState.textContent = 'Saved';
    savedTimer = window.setTimeout(() => { if (!hasChanges()) saveState.textContent = ''; }, 3000);
  } catch (error) { setFeedback(String(error?.message || error) || 'Could not save. Check configuration file permissions and try again.', 'error'); }
  finally { setBusy(false); }
});
async function loadProjects() {
  const result = await api.ListProjects();
  projects = Object.create(null);
  for (const [name, project] of Object.entries(result)) {
    projects[name] = { color: project.color || '', environments: Object.assign(Object.create(null), project.environments) };
  }
}
setBusy(true);
loadProjects().then(() => {
  ready = true; selectProject(Object.keys(projects).sort()[0] || ''); setBusy(false);
}).catch(() => { projectStatus.textContent = 'Could not load settings. Close and reopen Select setup.'; });
