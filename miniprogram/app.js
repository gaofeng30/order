const demoData = require("./mock/data");

App({
  globalData: {
    categories: demoData.categories,
    products: demoData.products,
    orders: demoData.orders,
    dashboard: demoData.dashboard,
    settings: demoData.settings,
    user: demoData.user,
    cart: [],
    lastOrderId: demoData.orders[0] ? demoData.orders[0].id : ""
  }
});
