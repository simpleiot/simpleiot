module Components.NodeAction exposing (view)

import Api.Node as Node
import Api.Point as Point
import Components.NodeOptions exposing (CopyMove(..), NodeOptions, findNode, oToInputO)
import Element exposing (..)
import Element.Background as Background
import Element.Border as Border
import Element.Font as Font
import UI.Icon as Icon
import UI.NodeInputs as NodeInputs
import UI.Style as Style
import UI.ViewIf exposing (viewIf)
import Html.Attributes exposing (disabled)


view : NodeOptions msg -> Element msg
view o =
    let
        icon =
            if o.node.typ == Node.typeAction then
                Icon.trendingUp

            else
                Icon.trendingDown

        active =
            Point.getBool o.node.points Point.typeActive "0"

        descBackgroundColor =
            if active then
                Style.colors.blue

            else
                Style.colors.none

        descTextColor =
            if active then
                Style.colors.white

            else
                Style.colors.black

        error =
            Point.getText o.node.points Point.typeError "0"

        disabled =
            Point.getBool o.node.points Point.typeDisabled ""

        titleBackground =
            if disabled then
                Style.colors.ltgray

            else if error /= "" then
                Style.colors.red

            else
                Style.colors.none
    in
    column
        [ width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color Style.colors.black
        , spacing 6
        ]
    <|
        wrappedRow
            [ spacing 10
            , paddingEach { top = 0, right = 10, bottom = 0, left = 0 }
            , Background.color titleBackground
            , width fill
            ]
            [ icon
            , el [ Background.color descBackgroundColor, Font.color descTextColor ] <|
                text <|
                    Point.getText o.node.points Point.typeDescription ""
            , if Point.getBool o.node.points Point.typeDisabled "" then
                text "(disabled)"
              else
                text ""
            ]
            :: (if o.expDetail then
                    let
                        labelWidth =
                            150

                        opts =
                            oToInputO o labelWidth

                        textInput =
                            NodeInputs.nodeTextInput opts "0"

                        optionInput =
                            NodeInputs.nodeOptionInput opts "0"

                        numberInput =
                            NodeInputs.nodeNumberInput opts "0"

                        actionType =
                            Point.getText o.node.points Point.typeAction "0"

                        actionSetValue =
                            actionType == Point.valueSetValue

                        actionPlayAudio =
                            actionType == Point.valuePlayAudio

                        valueType =
                            Point.getText o.node.points Point.typeValueType "0"

                        nodeId =
                            Point.getText o.node.points Point.typeNodeID "0"

                        checkboxInput =
                            NodeInputs.nodeCheckboxInput opts "0"
                    in
                    [ textInput Point.typeDescription "Description" ""
                    , checkboxInput Point.typeDisabled "Disabled"
                    , optionInput Point.typeAction
                        "Action"
                        [ ( Point.valueNotify, "notify" )
                        , ( Point.valueSetValue, "set node value" )
                        , ( Point.valuePlayAudio, "play audio" )
                        ]
                    , viewIf actionSetValue <|
                        optionInput Point.typePointType
                            "Point Type"
                            [ ( Point.typeValue, "value" )
                            , ( Point.typeValueSet, "set value (use for remote devices)" )
                            , ( Point.typeLightSet, "set light state" )
                            , ( Point.typeSwitchSet, "set switch state" )
                            ]
                    , viewIf actionSetValue <| textInput Point.typePointKey "Point Key" ""
                    , viewIf actionSetValue <| textInput Point.typeNodeID "Node ID" ""
                    , if nodeId /= "" then
                        let
                            nodeDesc =
                                case findNode o.nodes nodeId of
                                    Just node ->
                                        el [ Background.color Style.colors.ltblue ] <|
                                            text <|
                                                "("
                                                    ++ Node.getBestDesc node
                                                    ++ ")"

                                    Nothing ->
                                        el [ Background.color Style.colors.orange ] <| text "(node not found)"
                        in
                        el [ Font.italic, paddingEach { top = 0, right = 0, left = 170, bottom = 0 } ] <|
                            nodeDesc

                      else
                        Element.none
                    , viewIf actionSetValue <|
                        case o.copy of
                            CopyMoveNone ->
                                Element.none

                            Copy id _ desc ->
                                if nodeId /= id then
                                    let
                                        label =
                                            row
                                                [ spacing 10 ]
                                                [ text <| "paste ID for node: "
                                                , el
                                                    [ Font.italic
                                                    , Background.color Style.colors.ltblue
                                                    ]
                                                  <|
                                                    text desc
                                                ]
                                    in
                                    NodeInputs.nodePasteButton opts label Point.typeNodeID id

                                else
                                    Element.none
                    , viewIf actionSetValue <|
                        optionInput Point.typeValueType
                            "Point Value Type"
                            [ ( Point.valueNumber, "number" )
                            , ( Point.valueOnOff, "on/off" )
                            , ( Point.valueText, "text" )
                            ]
                    , viewIf actionSetValue <|
                        case valueType of
                            "number" ->
                                numberInput Point.typeValue "Value"

                            "onOff" ->
                                let
                                    onOffInput =
                                        NodeInputs.nodeOnOffInput opts "0"
                                in
                                onOffInput Point.typeValue Point.typeValue "Value"

                            "text" ->
                                textInput Point.typeValueText "Value" ""

                            _ ->
                                Element.none
                    , viewIf actionPlayAudio <|
                        textInput Point.typeDevice "Device" ""
                    , viewIf actionPlayAudio <|
                        numberInput Point.typeChannel "Channel"
                    , viewIf actionPlayAudio <|
                        textInput Point.typeFilePath "Wav file path" "/absolute/path/to/sound.wav"
                    , el [ Font.color Style.colors.red ] <| text error
                    , NodeInputs.nodeKeyValueInput opts Point.typeTag "Tags" "Add Tag"
                    ]

                else
                    []
               )
