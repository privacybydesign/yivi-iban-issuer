import React from 'react';
import {BrowserRouter, Routes, Route} from 'react-router-dom';

// Payment method components.
import Ideal from './Ideal';

import './App.css';

function App(props) {
  return (
    <>
      <a href="/">home</a>
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<Ideal />} />
        </Routes>
      </BrowserRouter>
    </>
  );
}

export default App;
