/**
 * 分页相关状态和事件
 */
import type { FilterType } from '@/typings/resource';

import { useResourceStore, useBusinessStore } from '@/store';
import { ref, onMounted, watch } from 'vue';
import { useWhereAmI } from '@/hooks/useWhereAmI';

type SortType = {
  column: {
    field: string;
  };
  type: string;
};

export type PropsType = {
  filter?: FilterType;
};

export default (
  props: PropsType,
  type: string,
  completeCallback: () => void,
  apiMethod?: Function,
  apiName = 'list',
  args: any = {},
  extraResolveData?: (...args: any) => Promise<any>, // 对表格数据做额外的处理
) => {
  // 接口
  const resourceStore = useResourceStore();
  const businessStore = useBusinessStore();

  // 查询列表相关状态
  const isLoading = ref(false);
  const datas = ref([]);
  const pagination = ref({
    current: 1,
    limit: 20,
    count: 0,
  });
  const sort = ref('created_at');
  const order = ref('DESC');

  // 连表查询时, sort 按照 created_at 字段排序时需要指定资源前缀
  switch (apiName) {
    case 'getUnbindCvmEips':
      sort.value = `eip.${sort.value}`;
      break;
    case 'getUnbindCvmDisks':
      sort.value = `disk.${sort.value}`;
      break;
    case 'getUnbindDiskCvms':
      sort.value = `cvm.${sort.value}`;
      break;
  }

  const isFilter = ref(false);
  const { isBusinessPage } = useWhereAmI();

  const targetStore = isBusinessPage ? businessStore : resourceStore;

  // 更新数据
  const triggerApi = () => {
    isLoading.value = true;
    // 默认拉取方法
    const getDefaultList = () =>
      Promise.all([
        targetStore[apiName](
          {
            page: {
              count: false,
              start: (pagination.value.current - 1) * pagination.value.limit,
              limit: pagination.value.limit,
              sort: sort.value,
              order: order.value,
            },
            filter: props.filter,
            ...args,
          },
          type,
        ),
        targetStore[apiName](
          {
            page: {
              count: true,
            },
            filter: props.filter,
            ...args,
          },
          type,
        ),
      ]);
    // 用户如果传了，就用传入的获取数据的方法
    const method = apiMethod || getDefaultList;
    // 执行获取数据的逻辑
    method()
      .then(([listResult, countResult]: [any, any]) => {
        datas.value = (listResult?.data?.details || listResult?.data || []).map((item: any) => {
          return {
            ...item,
            ...item.spec,
            ...item.attachment,
            ...item.revision,
            ...item.extension,
            ...item?.extension?.attachment,
          };
        });
        // 如果传入了 extraResolveData 方法，则执行该方法, 对 datas 做额外的处理
        typeof extraResolveData === 'function' && extraResolveData(datas.value).then((res) => (datas.value = res));
        pagination.value.count = countResult?.data?.count || 0;
      })
      .finally(() => {
        isLoading.value = false;
        isFilter.value = false;
        completeCallback?.();
      });
  };

  // 页码变化发生的事件
  const handlePageChange = (current: number) => {
    if (isFilter.value) return;
    pagination.value.current = current;
    triggerApi();
  };

  // 条数变化发生的事件
  const handlePageSizeChange = (limit: number) => {
    if (isFilter.value) return;
    pagination.value.limit = limit;
    triggerApi();
  };

  // 排序变化发生的事件
  const handleSort = ({ column, type }: SortType) => {
    pagination.value.current = 1;
    sort.value = column.field;
    order.value = type === 'desc' ? 'DESC' : 'ASC';
    triggerApi();
  };

  // 过滤发生变化的时候，获取数据
  watch(
    () => props.filter,
    () => {
      isFilter.value = true; // 如果是过滤则不需要再次请求
      pagination.value.current = 1; // 页码重置
      pagination.value.limit = 20;
      triggerApi();
    },
    {
      deep: true,
      // immediate: true,
    },
  );

  // 切换tab重新获取数据
  watch(
    () => type,
    () => {
      triggerApi();
    },
  );

  onMounted(triggerApi);

  return {
    datas,
    pagination,
    isLoading,
    handlePageChange,
    handlePageSizeChange,
    handleSort,
    triggerApi,
  };
};
