import {
  ClockCircleOutlined,
  MoreOutlined,
  PlayCircleOutlined,
  UserOutlined,
} from '@ant-design/icons';
import { Button, Dropdown } from 'antd';
import React from 'react';
import { TextEllipsis } from 'UI';

const backgroundUrl = '/assets/img/spotThumbBg.svg';

export function GridItem({
  title,
  onItemClick,
  thumbnail,
  setLoading,
  loading,
  user,
  createdAt,
  menuItems,
  onMenuClick,
  modifier,
}: {
  title: string;
  onItemClick: (e?: any) => void;
  thumbnail?: string;
  setLoading: (loading: boolean) => void;
  loading?: boolean;
  copyToClipboard: () => void;
  user: string;
  createdAt: string;
  menuItems: any[];
  onMenuClick: (key: any) => void;
  modifier: React.ReactNode;
}) {
  return (
    <div
      className="bg-white rounded-lg overflow-hidden shadow-xs border transition flex flex-col items-start hover:border-teal"
      data-test-id="highlight-grid-item"
    >
      <div
        className="relative group overflow-hidden"
        style={{
          width: '100%',
          height: 180,
          backgroundImage: `url(${backgroundUrl})`,
          backgroundSize: 'cover',
          backgroundPosition: 'center',
        }}
      >
        {loading && (
          <div className="absolute inset-0 flex items-center justify-center">
            {/* Reuse existing loader styling from UI via CSS; keep minimal here */}
            <div className="animate-spin w-6 h-6 rounded-full border-2 border-white border-t-transparent" />
          </div>
        )}
        <div
          className="block w-full h-full cursor-pointer transition hover:bg-teal/70 relative"
          onClick={onItemClick}
        >
          {thumbnail ? (
            <img
              src={thumbnail}
              alt={title}
              className="w-full h-full object-cover opacity-80"
              onLoad={() => setLoading(false)}
              onError={() => setLoading(false)}
              style={{ display: loading ? 'none' : 'block' }}
            />
          ) : null}
          <div className="absolute inset-0 flex items-center justify-center opacity-0 scale-75 transition-all hover:scale-100 group-hover:opacity-100">
            <PlayCircleOutlined
              style={{ fontSize: '48px', color: 'white' }}
              className="bg-teal/50 rounded-full"
            />
          </div>
        </div>

        {modifier}
      </div>
      <div className="w-full border-t">
        <div className="bg-yellow/50 mx-2 mt-2 px-2 w-full rounded-sm">
          <TextEllipsis text={title} className="capitalize" />
        </div>
        <div className="flex items-center gap-1 leading-4 text-xs opacity-50 p-3">
          <div>
            <UserOutlined />
          </div>
          <TextEllipsis text={user} />
          <div className="ml-auto">
            <ClockCircleOutlined />
          </div>
          <div>{createdAt}</div>
          <div>
            <Dropdown
              menu={{ items: menuItems, onClick: onMenuClick }}
              trigger={['click']}
            >
              <Button type="text" icon={<MoreOutlined />} size="small" />
            </Dropdown>
          </div>
        </div>
      </div>
    </div>
  );
}

