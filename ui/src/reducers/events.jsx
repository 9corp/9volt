import {createReducer} from '../utils';
import {EVENTS_BEGIN,EVENTS_SUCCESS,EVENTS_FAILURE} from '../actions/events';

const initialState = {
    isFetching: false,
    data: null,
    statusText: ""
};

export default createReducer(initialState,{
    [EVENTS_BEGIN]: (state,payload) => {
        return Object.assign({},state, {
            isFetching: true,
            statusText: "Retrieving events."
        });
    },
    [EVENTS_SUCCESS]: (state,payload) => {
        return Object.assign({},state, {
            isFetching: true,
            data: payload.data,
            StatusText: "Events retrieved."
        });
    },
    [EVENTS_FAILURE]: (state,payload) => {
        return Object.assign({},state, {
            isFetching: true,
            statusText: `Events Error: ${payload.status} ${payload.statusText}`
        });
    }
});