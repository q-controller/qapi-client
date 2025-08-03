import os
import shutil
import subprocess
from dataclasses import dataclass, field
from typing import List, Optional

from qapi.backend import QAPIBackend
from qapi.error import QAPISemError
from qapi.schema import (
    QAPISchema,
    QAPISchemaAlternatives,
    QAPISchemaBranches,
    QAPISchemaEnumMember,
    QAPISchemaFeature,
    QAPISchemaIfCond,
    QAPISchemaModule,
    QAPISchemaObjectType,
    QAPISchemaObjectTypeMember,
    QAPISchemaType,
    QAPISchemaVisitor,
)
from qapi.source import QAPISourceInfo
from utils import get_environment


@dataclass
class Enumeration:
    name: str
    values: list[str] = field(default_factory=list)


@dataclass
class Field:
    name: str
    typename: str
    optional: Optional[bool] = False


@dataclass
class Type:
    name: str
    fields: List[Field] = field(default_factory=list)


@dataclass
class Array:
    name: str
    element_type: str


@dataclass
class Method:
    name: str
    arg: Optional[str] = None
    ret: Optional[str] = None


@dataclass
class Event:
    name: str
    arg: Optional[str] = None


@dataclass
class Module:
    name: str
    types: list[Type] = field(default_factory=list)
    arrays: list[Array] = field(default_factory=list)
    enums: list[Enumeration] = field(default_factory=list)
    methods: list[Method] = field(default_factory=list)
    events: list[Event] = field(default_factory=list)


@dataclass
class Registry:
    modules: list[Module] = field(default_factory=list)


class QAPIGoVisitor(QAPISchemaVisitor):
    registry: Registry
    visited: set[str]

    def __init__(self):
        super().__init__()
        self.registry = Registry()
        self.visited = set()

    def visit_include(self, name: str, info: Optional[QAPISourceInfo]) -> None:
        if info:
            path = os.path.join(os.path.dirname(info.fname), name)
            if not path in self.visited:
                self.visited.add(path)
                try:
                    visitor = QAPIGoVisitor()
                    schema = QAPISchema(path)
                    schema.visit(visitor=visitor)
                    modules = set(map(lambda e: e.name, self.registry.modules))
                    for module in visitor.registry.modules:
                        if not module.name in modules:
                            self.registry.modules.insert(0, module)
                except QAPISemError as exc:
                    print(exc)

    def visit_module(self, name: str) -> None:
        self.registry.modules.append(Module(name=name))

    def visit_builtin_type(
        self, name: str, info: Optional[QAPISourceInfo], json_type: str
    ) -> None:
        self.registry.modules[-1].types.append(Type(name))

    def visit_enum_type(
        self,
        name: str,
        info: Optional[QAPISourceInfo],
        ifcond: QAPISchemaIfCond,
        features: List[QAPISchemaFeature],
        members: List[QAPISchemaEnumMember],
        prefix: Optional[str],
    ) -> None:
        enumeration = Enumeration(name=name)
        for m in members:
            enumeration.values.append(m.name)
        if self.registry.modules:
            self.registry.modules[-1].enums.append(enumeration)

    def visit_array_type(
        self,
        name: str,
        info: Optional[QAPISourceInfo],
        ifcond: QAPISchemaIfCond,
        element_type: QAPISchemaType,
    ) -> None:
        if self.registry.modules:
            self.registry.modules[-1].arrays.append(
                Array(name=name, element_type=element_type.name)
            )

    def visit_object_type_flat(
        self,
        name: str,
        info: Optional[QAPISourceInfo],
        ifcond: QAPISchemaIfCond,
        features: List[QAPISchemaFeature],
        members: List[QAPISchemaObjectTypeMember],
        branches: Optional[QAPISchemaBranches],
    ) -> None:
        obj = Type(name=name)
        if branches:
            for v in branches.variants:
                obj.fields.append(
                    Field(name=v.name, typename=v.type.name, optional=True)
                )
        for member in members:
            obj.fields.append(
                Field(
                    name=member.name,
                    typename=member.type.name,
                    optional=member.optional,
                )
            )
        if self.registry.modules:
            self.registry.modules[-1].types.append(obj)

    def visit_alternate_type(
        self,
        name: str,
        info: Optional[QAPISourceInfo],
        ifcond: QAPISchemaIfCond,
        features: List[QAPISchemaFeature],
        alternatives: QAPISchemaAlternatives,
    ) -> None:
        obj = Type(name=name)
        for v in alternatives.variants:
            obj.fields.append(Field(name=v.name, typename=v.type.name, optional=True))
        if self.registry.modules:
            self.registry.modules[-1].types.append(obj)

    def visit_command(
        self,
        name: str,
        info: Optional[QAPISourceInfo],
        ifcond: QAPISchemaIfCond,
        features: List[QAPISchemaFeature],
        arg_type: Optional[QAPISchemaObjectType],
        ret_type: Optional[QAPISchemaType],
        gen: bool,
        success_response: bool,
        boxed: bool,
        allow_oob: bool,
        allow_preconfig: bool,
        coroutine: bool,
    ) -> None:
        method = Method(name=name)
        method.ret = ret_type.name if ret_type else None
        method.arg = arg_type.name if arg_type else None
        if self.registry.modules:
            self.registry.modules[-1].methods.append(method)

    def visit_event(
        self,
        name: str,
        info: Optional[QAPISourceInfo],
        ifcond: QAPISchemaIfCond,
        features: List[QAPISchemaFeature],
        arg_type: Optional[QAPISchemaObjectType],
        boxed: bool,
    ) -> None:
        event = Event(name=name, arg=arg_type.name if arg_type else None)
        if self.registry.modules:
            self.registry.modules[-1].events.append(event)


class QAPIGoBackend(QAPIBackend):
    def generate(
        self,
        schema: QAPISchema,
        output_dir: str,
        prefix: str,
        unmask: bool,
        builtins: bool,
        gen_tracing: bool,
    ) -> None:
        pkg = prefix
        template_dir = os.path.dirname(os.path.abspath(__file__))
        env = get_environment(os.path.join(template_dir, "templates"))

        vis = QAPIGoVisitor()
        schema.visit(vis)

        dir = os.path.join(output_dir, pkg)
        os.makedirs(dir, exist_ok=True)

        modules: set[str] = set()
        for module in vis.registry.modules:
            if QAPISchemaModule.is_builtin_module(module.name):
                template = env.get_template("builtin.jinja2").render(
                    module=module, pkg=pkg
                )
                with open(os.path.join(dir, "builtin.go"), "w") as f:
                    f.write(template)
                continue
            moduleName = os.path.splitext(module.name)[0]
            template = env.get_template("module.jinja2")
            with open(os.path.join(dir, f"{moduleName}types.go"), "w") as f:
                f.write(template.render(module=module, pkg=pkg, moduleName=moduleName))
            if len(module.methods) > 0:
                modules.add(moduleName)
                template = env.get_template("service.go.jinja2")
                with open(os.path.join(dir, f"{moduleName}impl.go"), "w") as f:
                    f.write(
                        template.render(module=module, pkg=pkg, moduleName=moduleName)
                    )
        events: dict[str, Event] = {}
        for module in vis.registry.modules:
            for event in module.events:
                events.update({f"{event.name}": event})

        template = env.get_template("events.go.jinja2")
        with open(os.path.join(dir, "events.go"), "w") as f:
            f.write(template.render(events=events.values(), pkg=pkg))

        # optionally apply go fmt to all go files in dir if go compiler exists
        if shutil.which("go") is not None:
            try:
                for root, _, files in os.walk(dir):
                    for f in files:
                        if f.endswith(".go"):
                            subprocess.run(
                                ["go", "fmt", os.path.join(root, f)], check=True
                            )
            except subprocess.CalledProcessError as e:
                print(f"Error running go fmt: {e}")
