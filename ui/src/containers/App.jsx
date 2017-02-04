import React from 'react';
import Navigation from '../components/Nav';

export class App extends React.Component {

  render () {
    return (
        <div>
            <Navigation/>
            {this.props.children}
        </div>
    );
  }
}

export default App
