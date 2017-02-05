import {createReducer} from '../utils';
import {STATUS_BEGIN,STATUS_SUCCESS,STATUS_FAILURE} from '../actions/status';

const initialState = {
    isFetching: false,
    data: null,
    statusText: ""
};

export default createReducer(initialState,{
    [STATUS_BEGIN]: (state,payload) => {
        return Object.assign({},state, {
            isFetching: true,
            statusText: "Retrieving status."
        });
    },
    [STATUS_SUCCESS]: (state,payload) => {
        return Object.assign({},state, {
            isFetching: true,
            data: payload.data,
            statusText: "Status retrieved."
        });
    },
    [STATUS_FAILURE]: (state,payload) => {
        return Object.assign({},state, {
            isFetching: true,
            statusText: `Status Error: ${payload.status} ${payload.statusText}`
        });
    }
});