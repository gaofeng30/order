function clone(value) {
  return JSON.parse(JSON.stringify(value));
}

function pageMethods(definition) {
  const methods = {};
  for (const behavior of definition.behaviors || []) {
    Object.assign(methods, behavior.methods || {});
  }
  for (const [name, value] of Object.entries(definition)) {
    if (typeof value === 'function') methods[name] = value;
  }
  return methods;
}

function pageData(definition) {
  const data = {};
  for (const behavior of definition.behaviors || []) Object.assign(data, clone(behavior.data || {}));
  return Object.assign(data, clone(definition.data || {}));
}

function callLifecycle(component, definition, name, argument) {
  for (const behavior of definition.behaviors || []) {
    if (name === 'onLoad' && behavior.lifetimes && behavior.lifetimes.attached) {
      behavior.lifetimes.attached.call(component.instance);
    }
    if (typeof behavior[name] === 'function') behavior[name].call(component.instance, argument);
  }
  if (typeof definition[name] === 'function') return definition[name].call(component.instance, argument);
  return undefined;
}

function registerComponent({ definition, template, id, tagName, usingComponents = {} }) {
  return simulate.load(Object.assign({}, definition, {
    id,
    tagName,
    template,
    usingComponents,
  }));
}

function renderPage({ definition, template, id, usingComponents = {} }) {
  const componentID = simulate.load({
    id,
    template,
    data: pageData(definition),
    methods: pageMethods(definition),
    usingComponents,
  });
  const component = simulate.render(componentID);
  component.attach(document.body);
  callLifecycle(component, definition, 'onLoad', {});
  callLifecycle(component, definition, 'onShow');
  return component;
}

module.exports = { registerComponent, renderPage };
