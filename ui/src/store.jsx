import rootReducer from './reducers';
import thunk from 'redux-thunk';
import {applyMiddleware, compose, createStore} from 'redux';
import {routerMiddleware} from 'react-router-redux';
import createLogger from 'redux-logger';

export const setStore = (initialState,history) => {

    const routerMw = routerMiddleware(history);
    const middlewares = [routerMw,thunk]
    let composeEnhancers = compose

    if (process.env.NODE_ENV !== 'production') {
      const logger = createLogger();
      middlewares.push(logger);
      composeEnhancers = window.__REDUX_DEVTOOLS_EXTENSION_COMPOSE__
    }

    let createStoreWithMiddleware = composeEnhancers(
     applyMiddleware(...middlewares)
    );

    const store = createStoreWithMiddleware(createStore)(rootReducer, initialState);

    return store;
}