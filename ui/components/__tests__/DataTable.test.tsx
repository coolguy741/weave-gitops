import { fireEvent, render, screen } from "@testing-library/react";
import "jest-styled-components";
import React from "react";
import renderer from "react-test-renderer";
import { withTheme } from "../../lib/test-utils";
import DataTable, { SortType, sortWithType } from "../DataTable";

describe("DataTable", () => {
  const rows = [
    {
      name: "the-cool-app",
      status: "Ready",
      lastUpdate: "2005-01-02T15:04:05-0700",
      lastSyncedAt: 1000,
    },
    {
      name: "podinfo",
      status: "Failed",
      lastUpdate: "2006-01-02T15:04:05-0700",
      lastSyncedAt: 2000,
    },
    {
      name: "nginx",
      status: "Ready",
      lastUpdate: "2004-01-02T15:04:05-0700",
      lastSyncedAt: 3000,
    },
  ];

  const fields = [
    {
      label: "Name",
      value: ({ name }) => <a href="/some_url">{name}</a>,
      sortType: SortType.string,
      sortValue: ({ name }) => name,
    },
    {
      label: "Status",
      value: "status",
      sortType: SortType.bool,
      sortValue: ({ status }) => (status === "Ready" ? true : false),
    },
    {
      label: "Last Updated",
      value: "lastUpdate",
      sortType: SortType.date,
      sortValue: ({ lastUpdate }) => lastUpdate,
    },
    {
      label: "Last Synced At",
      value: "lastSyncedAt",
      sortType: SortType.number,
      sortValue: ({ lastSyncedAt }) => lastSyncedAt,
    },
  ];

  describe("sorting", () => {
    describe("sortWithType", () => {
      it("should handle sorting with case SortType.string", () => {
        const nameSort = sortWithType(rows, {
          label: "Name",
          value: ({ name }) => <a href="/some_url">{name}</a>,
          sortType: SortType.string,
          sortValue: ({ name }) => name,
        });
        expect(nameSort[0].name).toBe("nginx");
      });
      it("should handle sorting with case SortType.bool", () => {
        const boolSort = sortWithType(rows, {
          label: "Status",
          value: "status",
          sortType: SortType.bool,
          sortValue: ({ status }) => (status === "Ready" ? true : false),
        });
        expect(boolSort[0].status).toBe("Failed");
        expect(boolSort[2].status).toBe("Ready");
      });
      it("should handle sorting with case SortType.date", () => {
        const dateSort = sortWithType(rows, {
          label: "Last Updated",
          value: "lastUpdate",
          sortType: SortType.date,
          sortValue: ({ lastUpdate }) => lastUpdate,
        });
        expect(dateSort[0].lastUpdate).toBe("2004-01-02T15:04:05-0700");
      });
      it("should handle sorting with case SortType.number", () => {
        const numberSort = sortWithType(rows, {
          label: "Last Synced At",
          value: "lastSyncedAt",
          sortType: SortType.number,
          sortValue: ({ lastSyncedAt }) => lastSyncedAt,
        });
        expect(numberSort[0].lastSyncedAt).toBe(1000);
      });
    });
    it("initially sorts based on defaultSort", () => {
      render(withTheme(<DataTable fields={fields} rows={rows} />));
      const firstRow = screen.getAllByRole("row")[1];
      expect(firstRow.innerHTML).toMatch(/nginx/);
    });
    it("reverses sort on thead click", () => {
      render(withTheme(<DataTable fields={fields} rows={rows} />));

      const nameButton = screen.getByText("Name");
      fireEvent.click(nameButton);
      const firstRow = screen.getAllByRole("row")[1];
      expect(firstRow.innerHTML).toMatch(/the-cool-app/);
    });
    it("resets reverseSort and switches sort column on different thead click", () => {
      render(withTheme(<DataTable fields={fields} rows={rows} />));
      const nameButton = screen.getByText("Name");
      fireEvent.click(nameButton);
      const statusButton = screen.getByText("Status");
      fireEvent.click(statusButton);
      const firstRow = screen.getAllByRole("row")[1];
      expect(firstRow.innerHTML).toMatch(/podinfo/);
    });
    it("should render text when rows is empty", () => {
      render(withTheme(<DataTable fields={fields} rows={[]} />));
      const firstRow = screen.getAllByRole("row")[1];
      expect(firstRow.innerHTML).toMatch(/No/);
    });
  });
  describe("snapshots", () => {
    it("renders", () => {
      const tree = renderer
        .create(withTheme(<DataTable fields={fields} rows={rows} />))
        .toJSON();
      expect(tree).toMatchSnapshot();
    });
  });
});
