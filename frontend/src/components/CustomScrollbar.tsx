import * as React from 'react';
import { Scrollbars } from 'react-custom-scrollbars-2';
import { useTermTheme } from '@/theme/ThemeProvider';

type Props = {
  children: React.ReactNode;
  style?: React.CSSProperties;
  autoHide?: boolean;
  autoHideTimeout?: number;
  autoHideDuration?: number;
};

export default function CustomScrollbar({ 
  children, 
  style, 
  autoHide = false,
  autoHideTimeout = 1000,
  autoHideDuration = 200 
}: Props) {
  const { theme } = useTermTheme();

  const renderThumb = ({ style: thumbStyle, ...props }: any) => {
    return (
      <div
        {...props}
        style={{
          ...thumbStyle,
          backgroundColor: theme.scrollbarThumb || theme.border,
          borderRadius: '5px',
          cursor: 'pointer',
        }}
      />
    );
  };

  const renderTrack = ({ style: trackStyle, ...props }: any) => {
    return (
      <div
        {...props}
        style={{
          ...trackStyle,
          backgroundColor: 'transparent',
          right: '2px',
          bottom: '2px',
          top: '2px',
          borderRadius: '3px',
          width: '10px',
        }}
      />
    );
  };

  return (
    <Scrollbars
      style={style}
      autoHide={autoHide}
      autoHideTimeout={autoHideTimeout}
      autoHideDuration={autoHideDuration}
      renderThumbVertical={renderThumb}
      renderTrackVertical={renderTrack}
      renderView={(props) => (
        <div {...props} style={{ 
          ...props.style, 
          overflowX: 'hidden', 
          overflowY: 'scroll' }} />
      )}
    >
      {children}
    </Scrollbars>
  );
}
