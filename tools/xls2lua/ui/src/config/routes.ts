export default [
  {
    path: '/exportTable',
    name: "export-table",
    icon: "smile",
    component: "./exportTable",
  },
  {
    path: '/',
    redirect: '/exportTable',
  },
  {
    component: './404',
  },
];
