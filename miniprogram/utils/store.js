function appData() {
  return getApp().globalData;
}

function money(value) {
  return Number(value || 0).toFixed(2);
}

function getProduct(id) {
  return appData().products.find((item) => item.id === id);
}

function getProductsByCategory(categoryId) {
  return appData().products.filter((item) => item.categoryId === categoryId);
}

function getCart() {
  return appData().cart.map((item) => {
    const product = getProduct(item.productId);
    return {
      ...item,
      product,
      subtotal: product ? product.price * item.quantity : 0
    };
  });
}

function getCartSummary() {
  const items = getCart();
  const totalQuantity = items.reduce((sum, item) => sum + item.quantity, 0);
  const totalAmount = items.reduce((sum, item) => sum + item.subtotal, 0);

  return {
    items,
    totalQuantity,
    totalAmount,
    totalAmountText: money(totalAmount)
  };
}

function addToCart(productId, step) {
  const product = getProduct(productId);
  if (!product || product.status !== "available") {
    return getCartSummary();
  }

  const data = appData();
  const target = data.cart.find((item) => item.productId === productId);
  const nextStep = step || 1;

  if (target) {
    target.quantity = Math.max(0, target.quantity + nextStep);
    if (target.quantity === 0) {
      data.cart = data.cart.filter((item) => item.productId !== productId);
    }
  } else if (nextStep > 0) {
    data.cart.push({ productId, quantity: nextStep });
  }

  return getCartSummary();
}

function clearCart() {
  appData().cart = [];
  return getCartSummary();
}

function createOrder(form) {
  const data = appData();
  const summary = getCartSummary();
  const nextIndex = data.orders.length + 1;
  const pickupNo = `A${String(120 + nextIndex).padStart(3, "0")}`;
  const order = {
    id: `OD20260503${String(nextIndex + 1).padStart(3, "0")}`,
    pickupNo,
    status: "ready",
    statusText: "待取餐",
    amount: summary.totalAmount,
    createdAt: "刚刚",
    contact: form.contact || data.user.name,
    phone: form.phone || data.user.phone,
    remark: form.remark || "无备注",
    tastes: form.tastes || [],
    tasteText: form.tastes && form.tastes.length ? form.tastes.join("、") : "无特殊要求",
    pickupPlace: data.settings.pickupPlace,
    pickupTime: "预计 12:10-12:25",
    items: summary.items.map((item) => ({
      productId: item.productId,
      name: item.product.name,
      price: item.product.price,
      quantity: item.quantity
    }))
  };

  data.orders.unshift(order);
  data.lastOrderId = order.id;
  data.dashboard.orders += 1;
  data.dashboard.revenue += order.amount;
  clearCart();

  return order;
}

function getOrders(status) {
  const orders = appData().orders;
  if (!status || status === "all") {
    return orders;
  }
  return orders.filter((order) => order.status === status);
}

function getOrder(orderId) {
  return appData().orders.find((order) => order.id === orderId) || appData().orders[0];
}

function verifyOrder(orderId) {
  const order = getOrder(orderId);
  if (!order) {
    return null;
  }
  order.status = "completed";
  order.statusText = "已完成";
  order.pickupTime = "刚刚完成核销";
  return order;
}

function getDashboard() {
  const data = appData();
  return {
    ...data.dashboard,
    revenueText: money(data.dashboard.revenue),
    latestOrders: data.orders.slice(0, 3)
  };
}

module.exports = {
  money,
  getProduct,
  getProductsByCategory,
  getCartSummary,
  addToCart,
  clearCart,
  createOrder,
  getOrders,
  getOrder,
  verifyOrder,
  getDashboard
};
