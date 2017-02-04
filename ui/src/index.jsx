import React from 'react';
import ReactDOM from 'react-dom';
import {Provider} from 'react-redux';
import { browserHistory, Router, Route, IndexRoute } from 'react-router';
import { syncHistoryWithStore } from 'react-router-redux'
import NavBar from './navbar';
import {setStore} from './store';
import {HomeView} from './views';

// import "./styles/bootstrap.min.css

const store = setStore(window.__INITIAL_STATE__,browserHistory);
const history = syncHistoryWithStore(browserHistory,store);

const App = ({children}) => {
    return (
        <div>
            <NavBar />
            {children}
        </div>
    );
}
    
const Root = ({store,history}) => {
    return (
        <Provider store={store}>
            <Router history={history}>
                <Route component={App}>
                    <Route path="/">
                        <IndexRoute component={HomeView}/>
                    </Route>
                </Route>
            </Router>
        </Provider>
    );
}

ReactDOM.render(<Root store={store} history={history}/>,document.getElementById('root'));
