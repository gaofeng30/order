class ReservationMenuError extends Error {
  constructor(code) {
    super(code === 'INVALID_MENU_SELECTION' ? 'invalid menu selection' : 'menu unavailable');
    this.name = 'ReservationMenuError';
    this.code = code;
  }
}

function unavailable() {
  return new ReservationMenuError('MENU_UNAVAILABLE');
}

function getMenu(date, time) {
  return new Promise((resolve, reject) => {
    try {
      const baseUrl = getApp().globalData.apiBaseUrl;
      wx.request({
        url: `${baseUrl}/api/v1/menu?date=${encodeURIComponent(date)}&time=${encodeURIComponent(time)}`,
        method: 'GET',
        success(response) {
          if (response.statusCode === 200) {
            resolve(response.data);
            return;
          }
          if (
            response.statusCode === 400
            && response.data
            && response.data.error
            && response.data.error.code === 'INVALID_MENU_SELECTION'
          ) {
            reject(new ReservationMenuError('INVALID_MENU_SELECTION'));
            return;
          }
          reject(unavailable());
        },
        fail() { reject(unavailable()); },
      });
    } catch (error) {
      reject(unavailable());
    }
  });
}

module.exports = { ReservationMenuError, getMenu };
