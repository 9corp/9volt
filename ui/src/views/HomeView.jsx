import React from 'react';
import ReactDOM from 'react-dom';
import {connect} from 'react-redux';

class HomeView extends React.Component {

  render () {
    return (
      <div>
        HomeView
      </div>
    );
  }
}

export default connect()(HomeView)
