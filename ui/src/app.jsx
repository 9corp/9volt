import React from 'react';
import NavBar from './navbar';
import {Grid} from 'semantic-ui-react';

export const App = ({children}) => {
    return (
        <div>
            <div style={styles.navbar}>
                <NavBar />
            </div>
            <div style={styles.main}>
                {children}
            </div>
        </div>
    );
}

const menuWidth = 85;
const styles = {
    navbar: {
        position: 'fixed',
        top: 0,
        bottom: 0,
        left: 0,
        width: menuWidth,
        paddingBottom: '1em'
    },
    main: {
        marginLeft: 100,
        marginTop: 10,
        marginRight: 20,
        minWidth: parseInt(menuWidth, 20) + 300,
        maxWidth: parseInt(menuWidth, 20) + 900
    }
}