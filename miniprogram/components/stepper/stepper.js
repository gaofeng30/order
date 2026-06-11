Component({
  properties: {
    value: { type: Number, value: 0 },
    blue: { type: Boolean, value: false },
  },
  methods: {
    onSub() { this.triggerEvent('sub'); },
    onAdd() { this.triggerEvent('add'); },
  },
});
