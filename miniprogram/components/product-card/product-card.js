Component({
  properties: {
    product: {
      type: Object,
      value: {}
    },
    quantity: {
      type: Number,
      value: 0
    }
  },
  methods: {
    openDetail() {
      this.triggerEvent("detail", { id: this.data.product.id });
    },
    add() {
      this.triggerEvent("add", { id: this.data.product.id });
    },
    minus() {
      this.triggerEvent("minus", { id: this.data.product.id });
    }
  }
});
