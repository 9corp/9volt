export const STATUS_BEGIN = 'STATUS_BEGIN'
export const STATUS_SUCCESS = 'STATUS_SUCCESS'
export const STATUS_FAILURE = 'STATUS_FAILURE'

export const getStatus = () => (dispatch) => _getStatus(dispatch)

const _statusBegin = () => ({type: STATUS_BEGIN})
const _statusSuccess = (data) => ({type: STATUS_SUCCESS, payload: { data: data }})
const _statusFailure = (error) => ({type: STATUS_FAILURE, payload: error})

const _getStatus = (dispatch) => {
  dispatch(_statusBegin());
  return fetch('/status/check',
    {
      method: "GET",
      headers: {'Content-Type': 'application/json'}
    })
    .then(response => {
      if (response.status >= 200 && response.status < 300) {
          return response
      } else {
          let error = new Error(response.statusText);
          error.response = response;
          throw error
      }
    })
    .then(response => {
      return response.json()
    })
    .then(response => {
          dispatch(_statusSuccess(response));
    })
    .catch(error => {
        if(!error.response) {
          dispatch(_statusFailure({ status: 500, statusText: error.message}));
        } else if(error.response.status !== 200) {
          dispatch(_statusFailure({ status: error.response.status, statusText: error}));
        }
    });
}