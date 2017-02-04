import React from 'react';
import {Route, IndexRoute} from 'react-router';
import {App} from '../containers';
import {HomeView} from '../views';

export default(
    <Route component={App}>
      <Route path="/">
        <IndexRoute component={HomeView}/>
      </Route>
    </Route>
);
