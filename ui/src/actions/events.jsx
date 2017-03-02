export const EVENTS_BEGIN = 'EVENTS_BEGIN'
export const EVENTS_SUCCESS = 'EVENTS_SUCCESS'
export const EVENTS_FAILURE = 'EVENTS_FAILURE'

export const getEvents = () => (dispatch) => _getEvents(dispatch)

const _eventsBegin = () => ({type: EVENTS_BEGIN})
const _eventsSuccess = (data) => ({type: EVENTS_SUCCESS, payload: { data: data }})
const _eventsFailure = (error) => ({type: EVENTS_FAILURE, payload: error})

const _getEvents = (dispatch) => {
  dispatch(_eventsBegin());
  return fetch('/api/v1/event',
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
          dispatch(_eventsSuccess(response));
    })
    .catch(error => {
        if(!error.response) {
          dispatch(_eventsFailure({ status: 500, statusText: error.message}));
        } else if(error.response.status !== 200) {
          dispatch(_eventsFailure({ status: error.response.status, statusText: error}));
        }
    });
}