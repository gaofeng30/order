// This production package stays on the production backend in preview, trial,
// and release. UAT is built independently from the uat branch.
const origin = 'https://order.suianwuye.com';
module.exports = { develop: origin, trial: origin, release: origin };
