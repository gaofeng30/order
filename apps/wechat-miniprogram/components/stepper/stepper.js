Component({
  properties: {
    value: { type: Number, value: 0 },
  },
  methods: {
    onSub() { this.triggerEvent('sub'); },
    onAdd() { this.triggerEvent('add'); },
  },
});
