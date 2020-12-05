module Components.NodeModbusIO exposing (view)

import Api.Node exposing (Node)
import Api.Point as Point exposing (Point)
import Element exposing (..)
import Element.Border as Border
import Time
import UI.Form as Form
import UI.Icon as Icon
import UI.Style exposing (colors)
import UI.ViewIf exposing (viewIf)


view :
    { isRoot : Bool
    , now : Time.Posix
    , zone : Time.Zone
    , modified : Bool
    , expDetail : Bool
    , parent : Maybe Node
    , node : Node
    , onApiDelete : String -> msg
    , onEditNodePoint : String -> Point -> msg
    , onDiscardEdits : msg
    , onApiPostPoints : String -> msg
    }
    -> Element msg
view o =
    let
        textInput =
            Form.nodeTextInput { onEditNodePoint = o.onEditNodePoint, node = o.node, now = o.now }

        numberInput =
            Form.nodeNumberInput { onEditNodePoint = o.onEditNodePoint, node = o.node, now = o.now }

        optionInput =
            Form.nodeOptionInput { onEditNodePoint = o.onEditNodePoint, node = o.node, now = o.now }

        modbusIOType =
            Point.getPointText o.node.points Point.typeModbusIOType

        isClient =
            case o.parent of
                Just p ->
                    Point.getPointText p.points Point.typeClientServer == Point.valueClient

                Nothing ->
                    False
    in
    column
        [ width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color colors.black
        , spacing 6
        ]
    <|
        wrappedRow [ spacing 10 ]
            [ Icon.io
            , text <|
                Point.getPointText o.node.points Point.typeDescription
                    ++ ": "
                    ++ String.fromFloat
                        (Point.getPointValue o.node.points Point.typeValue)
                    ++ (if modbusIOType == Point.valueModbusRegister then
                            " " ++ Point.getPointText o.node.points Point.typeUnits

                        else
                            ""
                       )
            ]
            :: (if o.expDetail then
                    [ textInput Point.typeDescription "Description"
                    , viewIf isClient <| numberInput Point.typeID "ID"
                    , numberInput Point.typeAddress "Address"
                    , optionInput Point.typeModbusIOType
                        "IO type"
                        [ ( Point.valueModbusInput, "input" )
                        , ( Point.valueModbusCoil, "coil" )
                        , ( Point.valueModbusRegister, "register" )
                        ]
                    , viewIf (modbusIOType == Point.valueModbusRegister) <|
                        numberInput Point.typeScale "Scale factor"
                    , viewIf (modbusIOType == Point.valueModbusRegister) <|
                        numberInput Point.typeOffset "Offset"
                    , viewIf (modbusIOType == Point.valueModbusRegister) <|
                        textInput Point.typeUnits "Units"
                    , viewIf (modbusIOType == Point.valueModbusRegister) <|
                        optionInput Point.typeDataFormat
                            "Data format"
                            [ ( Point.valueUINT16, "UINT16" )
                            , ( Point.valueINT16, "INT16" )
                            , ( Point.valueUINT32, "UINT32" )
                            , ( Point.valueINT32, "INT32" )
                            , ( Point.valueFLOAT32, "FLOAT32" )
                            ]
                    , numberInput Point.typeValue "Value"
                    , viewIf o.modified <|
                        Form.buttonRow
                            [ Form.button
                                { label = "save"
                                , color = colors.blue
                                , onPress = o.onApiPostPoints o.node.id
                                }
                            , Form.button
                                { label = "discard"
                                , color = colors.gray
                                , onPress = o.onDiscardEdits
                                }
                            ]
                    ]

                else
                    []
               )
